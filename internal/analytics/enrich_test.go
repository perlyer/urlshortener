package analytics

import (
	"testing"
	"time"

	"github.com/perlyer/urlshortener/internal/events"
)

func TestExtractDimensions(t *testing.T) {
	e := events.ClickEvent{
		Code:      "abc",
		Timestamp: time.Date(2026, 7, 23, 10, 30, 0, 0, time.UTC),
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
			"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Referer:  "https://www.google.com/search?q=test",
		Language: "ru",
		IP:       "127.0.0.1",
	}

	got := map[string]string{}
	for _, d := range extractDimensions(e, nil) { // geo=nil → country "unknown"
		got[d.Name] = d.Value
	}

	want := map[string]string{
		"total":    "all",
		"hour":     "2026-07-23T10",
		"device":   "desktop",
		"browser":  "Chrome",
		"language": "ru",
		"referer":  "www.google.com",
		"country":  "unknown",
	}
	for name, exp := range want {
		if got[name] != exp {
			t.Errorf("измерение %q = %q, ожидали %q", name, got[name], exp)
		}
	}
	if got["os"] == "" || got["os"] == "unknown" {
		t.Errorf("os не определён: %q", got["os"])
	}
}

func TestRefererDirect(t *testing.T) {
	e := events.ClickEvent{Timestamp: time.Now(), Referer: ""}
	for _, d := range extractDimensions(e, nil) {
		if d.Name == "referer" && d.Value != "direct" {
			t.Fatalf("пустой referer → %q, ожидали direct", d.Value)
		}
	}
}
