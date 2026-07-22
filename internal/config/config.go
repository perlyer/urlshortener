// Package config читает настройки сервисов из переменных окружения.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config - общие настройки для api и redirector.
type Config struct {
	DatabaseURL string // строка подключения к Postgres
	Port        string // порт HTTP-сервера
	BaseURL     string // базовый адрес коротких ссылок (хост redirector)
	CodeLength  int    // длина генерируемого короткого кода
}

// Load собирает конфиг из окружения. DATABASE_URL обязателен, остальное
// имеет разумные значения по умолчанию.
func Load() (Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        getenv("PORT", "8080"),
		BaseURL:     getenv("BASE_URL", "http://localhost:8081"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: переменная DATABASE_URL обязательна")
	}

	codeLen, err := strconv.Atoi(getenv("CODE_LENGTH", "7"))
	if err != nil {
		return Config{}, fmt.Errorf("config: CODE_LENGTH должен быть числом: %w", err)
	}
	cfg.CodeLength = codeLen

	return cfg, nil
}

// getenv возвращает значение переменной окружения или fallback, если пусто.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
