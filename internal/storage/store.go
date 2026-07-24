// Package storage - слой доступа к хранилищу ссылок: создание с ретраями на
// коллизию кода и резолв кода в оригинальный URL. Это «репозиторий»: только
// здесь живёт знание о том, как ссылка попадает в БД и достаётся из неё.
package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/perlyer/urlshortener/internal/metrics"
	"github.com/perlyer/urlshortener/internal/shortcode"
	"github.com/perlyer/urlshortener/internal/storage/db"
)

// maxAttempts - сколько раз пытаемся подобрать свободный код при коллизии.
const maxAttempts = 5

// Ошибки, которые слой отдаёт наружу. Вызывающий проверяет их через
// errors.Is, ничего не зная про Postgres.
var (
	ErrNotFound      = errors.New("storage: ссылка не найдена")
	ErrCodeExhausted = errors.New("storage: не удалось подобрать свободный код")
)

// Cache - интерфейс кэша ссылок, нужный Store. Реализуется пакетом cache
// (RedisCache). Определён здесь, у потребителя, - это идиома Go: интерфейс
// описывает того, кто им пользуется, а не того, кто его реализует.
type Cache interface {
	Get(ctx context.Context, key string) (value string, found bool, err error)
	Set(ctx context.Context, key, value string) error
}

// Store оборачивает sqlc-запросы и кэш, хранит длину кода.
type Store struct {
	q       *db.Queries
	cache   Cache
	codeLen int
}

// New собирает Store поверх *db.Queries и кэша.
func New(q *db.Queries, cache Cache, codeLen int) *Store {
	return &Store{q: q, cache: cache, codeLen: codeLen}
}

// CreateShortLink генерирует случайный код и вставляет ссылку. Если код уже
// занят (нарушение UNIQUE), пробует другой - до maxAttempts раз.
func (s *Store) CreateShortLink(ctx context.Context, originalURL string) (db.Link, error) {
	for i := 0; i < maxAttempts; i++ {
		code, err := shortcode.Generate(s.codeLen)
		if err != nil {
			return db.Link{}, fmt.Errorf("storage: генерация кода: %w", err)
		}

		link, err := s.q.CreateLink(ctx, db.CreateLinkParams{
			Code:        code,
			OriginalURL: originalURL,
		})
		if err == nil {
			_ = s.cache.Set(ctx, "url:"+code, originalURL)
			return link, nil // успех
		}

		// Разворачиваем ошибку в тип pgconn.PgError и смотрим SQL-код.
		// 23505 = unique_violation → код занят, пробуем следующий.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			continue
		}
		// Любая другая ошибка БД - отдаём наверх.
		return db.Link{}, fmt.Errorf("storage: вставка ссылки: %w", err)
	}
	return db.Link{}, ErrCodeExhausted
}

// Resolve возвращает оригинальный URL по коду или ErrNotFound, если кода нет.
func (s *Store) Resolve(ctx context.Context, code string) (string, error) {
	url := "url:" + code
	value, found, err := s.cache.Get(ctx, url)
	if found {
		metrics.CacheHit()
		return value, nil
	}
	metrics.CacheMiss()

	link, err := s.q.GetLink(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("storage: поиск ссылки: %w", err)
	}

	_ = s.cache.Set(ctx, url, link.OriginalURL)
	return link.OriginalURL, nil
}

// ClickStats возвращает все агрегаты аналитики по коду (строки вида
// dimension/value/count).
func (s *Store) ClickStats(ctx context.Context, code string) ([]db.ListClickStatsRow, error) {
	rows, err := s.q.ListClickStats(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("storage: статистика: %w", err)
	}
	return rows, nil
}
