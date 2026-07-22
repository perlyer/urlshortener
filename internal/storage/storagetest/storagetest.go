// Package storagetest поднимает одноразовый Postgres в Docker для интеграционных
// тестов и отдаёт готовый *storage.Store. Вынесен в отдельный пакет, чтобы им
// могли пользоваться и тесты storage, и тесты сервисов в cmd/.
package storagetest

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // регистрирует драйвер "pgx" для database/sql
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/perlyer/urlshortener/internal/storage"
	"github.com/perlyer/urlshortener/internal/storage/db"
)

// NewStore запускает Postgres в контейнере, накатывает миграции проекта и
// возвращает готовый Store. Контейнер и пул соединений закрываются
// автоматически по завершении теста (через t.Cleanup).
func NewStore(t *testing.T, ctx context.Context) *storage.Store {
	t.Helper()

	pg, err := postgres.Run(ctx, "postgres:17",
		postgres.WithDatabase("shortener"),
		postgres.WithUsername("shortener"),
		postgres.WithPassword("shortener"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("storagetest: не удалось поднять postgres: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })

	connStr, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("storagetest: connection string: %v", err)
	}

	// Миграции накатываем goose'ом поверх database/sql (драйвер pgx stdlib),
	// используя те же файлы из migrations/, что и в проде - единый источник схемы.
	sqlDB, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("storagetest: sql.Open: %v", err)
	}
	goose.SetDialect("postgres")
	if err := goose.Up(sqlDB, migrationsDir()); err != nil {
		t.Fatalf("storagetest: миграции: %v", err)
	}
	_ = sqlDB.Close()

	// Для самого Store используем родной pgx-пул.
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("storagetest: pgxpool: %v", err)
	}
	t.Cleanup(pool.Close)

	return storage.New(db.New(pool), 7)
}

// migrationsDir отдаёт абсолютный путь к app/migrations относительно этого
// файла, чтобы хелпер работал из тестов любого пакета независимо от cwd.
func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "migrations")
}
