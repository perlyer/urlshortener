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

// Store оборачивает сгенерённые sqlc-запросы и хранит длину кода.
type Store struct {
	q       *db.Queries
	codeLen int
}

// New собирает Store поверх готового *db.Queries.
func New(q *db.Queries, codeLen int) *Store {
	return &Store{q: q, codeLen: codeLen}
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
	link, err := s.q.GetLink(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("storage: поиск ссылки: %w", err)
	}
	return link.OriginalURL, nil
}
