package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/perlyer/urlshortener/internal/storage/storagetest"
)

func TestCreateLinkHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s := storagetest.NewStore(t, ctx)

	h := makeCreateHandler(s, "http://localhost:8081")
	req := httptest.NewRequest(http.MethodPost, "/api/links",
		bytes.NewBufferString(`{"url":"https://example.com"}`))
	rec := httptest.NewRecorder()

	h(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("статус %d, ожидали 201; тело: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Code     string `json:"code"`
		ShortURL string `json:"short_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("невалидный JSON в ответе: %v", err)
	}
	if resp.Code == "" {
		t.Fatal("пустой code в ответе")
	}
	if resp.ShortURL != "http://localhost:8081/"+resp.Code {
		t.Fatalf("short_url = %q не соответствует коду %q", resp.ShortURL, resp.Code)
	}
}

func TestCreateLinkRejectsEmptyURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s := storagetest.NewStore(t, ctx)

	h := makeCreateHandler(s, "http://localhost:8081")
	req := httptest.NewRequest(http.MethodPost, "/api/links",
		bytes.NewBufferString(`{"url":""}`))
	rec := httptest.NewRecorder()

	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("статус %d, ожидали 400", rec.Code)
	}
}
