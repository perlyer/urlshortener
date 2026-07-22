package shortcode

import (
	"crypto/rand"
	"fmt"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func Generate(n int) (string, error) {
	buf := make([]byte, n)
	for i := 0; i < n; {
		var b [1]byte
		if _, err := rand.Read(b[:]); err != nil {
			return "", fmt.Errorf("shortcode: чтение энтропии: %w", err)
		}
		if b[0] >= 248 {
			continue
		}
		buf[i] = alphabet[b[0]%62]
		i++
	}
	return string(buf), nil
}
