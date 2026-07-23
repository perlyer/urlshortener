package events

import (
	"encoding/json"
	"testing"
	"time"
)

// Событие должно переживать сериализацию в JSON и обратно без потерь -
// это контракт между producer (redirector) и consumer (analytics).
func TestClickEventRoundTrip(t *testing.T) {
	e := ClickEvent{
		Code:      "abc123",
		Timestamp: time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC),
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
		Referer:   "https://google.com",
		Language:  "ru",
		IP:        "1.2.3.4",
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ClickEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Code != e.Code || got.UserAgent != e.UserAgent ||
		got.Referer != e.Referer || got.Language != e.Language || got.IP != e.IP {
		t.Fatalf("строковые поля разошлись:\n got %+v\nwant %+v", got, e)
	}
	if !got.Timestamp.Equal(e.Timestamp) {
		t.Fatalf("timestamp: got %v, want %v", got.Timestamp, e.Timestamp)
	}
}
