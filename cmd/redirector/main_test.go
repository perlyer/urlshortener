package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/perlyer/urlshortener/internal/storage/storagetest"
)

func TestRedirectFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s := storagetest.NewStore(t, ctx)

	link, err := s.CreateShortLink(ctx, "https://example.com/target")
	if err != nil {
		t.Fatalf("подготовка: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/{code}", makeRedirectHandler(s))

	req := httptest.NewRequest(http.MethodGet, "/"+link.Code, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("статус %d, ожидали 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "https://example.com/target" {
		t.Fatalf("Location = %q", loc)
	}
}

func TestRedirectNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s := storagetest.NewStore(t, ctx)

	r := chi.NewRouter()
	r.Get("/{code}", makeRedirectHandler(s))

	req := httptest.NewRequest(http.MethodGet, "/nosuchcode", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("статус %d, ожидали 404", rec.Code)
	}
}
