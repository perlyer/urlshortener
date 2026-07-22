package storage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/perlyer/urlshortener/internal/storage"
	"github.com/perlyer/urlshortener/internal/storage/storagetest"
)

func TestCreateAndResolve(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s := storagetest.NewStore(t, ctx)

	link, err := s.CreateShortLink(ctx, "https://example.com/very/long")
	if err != nil {
		t.Fatalf("CreateShortLink: %v", err)
	}
	if link.Code == "" {
		t.Fatal("код пустой")
	}

	got, err := s.Resolve(ctx, link.Code)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "https://example.com/very/long" {
		t.Fatalf("Resolve вернул %q, ожидали оригинальный URL", got)
	}
}

func TestResolveNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s := storagetest.NewStore(t, ctx)

	_, err := s.Resolve(ctx, "nosuchcode")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("ожидали ErrNotFound, получили %v", err)
	}
}
