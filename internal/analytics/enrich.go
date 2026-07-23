// Package analytics раскладывает события кликов на измерения и агрегирует их.
package analytics

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/url"

	"github.com/mssola/user_agent"
	"github.com/oschwald/geoip2-golang"

	"github.com/perlyer/urlshortener/internal/events"
)

// Dimension - одно измерение аналитики: имя (browser, device, …) и значение.
type Dimension struct {
	Name  string
	Value string
}

// extractDimensions раскладывает клик на измерения для агрегации. geo может
// быть nil (тогда страна - "unknown"), это удобно для тестов.
func extractDimensions(e events.ClickEvent, geo *geoip2.Reader) []Dimension {
	ua := user_agent.New(e.UserAgent)
	browser, _ := ua.Browser()

	return []Dimension{
		{"total", "all"},
		{"hour", e.Timestamp.UTC().Format("2006-01-02T15")},
		{"device", deviceType(ua)},
		{"browser", orUnknown(browser)},
		{"os", orUnknown(ua.OS())},
		{"language", orUnknown(e.Language)},
		{"referer", refererHost(e.Referer)},
		{"country", country(geo, e.IP)},
	}
}

func deviceType(ua *user_agent.UserAgent) string {
	switch {
	case ua.Bot():
		return "bot"
	case ua.Mobile():
		return "mobile"
	default:
		return "desktop"
	}
}

// refererHost вытаскивает домен источника; пустой referer → "direct".
func refererHost(ref string) string {
	if ref == "" {
		return "direct"
	}
	u, err := url.Parse(ref)
	if err != nil || u.Host == "" {
		return "direct"
	}
	return u.Host
}

// country определяет ISO-код страны по IP через локальную базу GeoIP.
func country(geo *geoip2.Reader, ipStr string) string {
	if geo == nil {
		return "unknown"
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "unknown"
	}
	rec, err := geo.Country(ip)
	if err != nil || rec.Country.IsoCode == "" {
		return "unknown"
	}
	return rec.Country.IsoCode
}

// visitorHash - приватный отпечаток посетителя (хэш IP+UA), чтобы считать
// уникальных, не храня сырой IP.
func visitorHash(ip, ua string) string {
	sum := sha256.Sum256([]byte(ip + "|" + ua))
	return hex.EncodeToString(sum[:8])
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
