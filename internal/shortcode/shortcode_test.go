package shortcode

import (
	"strings"
	"testing"
)

func TestGenerateLength(t *testing.T) {
	for _, n := range []int{1, 5, 7, 12} {
		code, err := Generate(n)
		if err != nil {
			t.Fatalf("Generate(%d) вернул ошибку: %v", n, err)
		}
		if len(code) != n {
			t.Fatalf("Generate(%d): длина %d, ожидали %d", n, len(code), n)
		}
	}
}

func TestGenerateAlphabet(t *testing.T) {
	const allowed = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	code, err := Generate(64)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	for _, r := range code {
		if !strings.ContainsRune(allowed, r) {
			t.Fatalf("символ %q вне base62-алфавита", r)
		}
	}
}

func TestGenerateRandomish(t *testing.T) {
	// Два вызова подряд практически никогда не совпадают.
	a, _ := Generate(10)
	b, _ := Generate(10)
	if a == b {
		t.Fatalf("два вызова дали одинаковый код %q - генератор не случайный", a)
	}
}
