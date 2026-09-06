package config

import (
	"log/slog"
	"testing"
	"time"
)

// Table-driven тест: список случаев + один цикл, который прогоняет все.
// Новый случай = новая строка в срезе tests.
func TestParseLevel(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"регистр не важен", "ERROR", slog.LevelError},
		{"мусор падает в info", "whatever", slog.LevelInfo},
		{"пустая строка падает в info", "", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLevel(tt.in)
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("без DATABASE_URL возвращает ошибку", func(t *testing.T) {
		// на всякий случай гасим переменную, если она пришла из реального окружения
		t.Setenv("DATABASE_URL", "")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() = nil error, want error про DATABASE_URL")
		}
	})

	t.Run("необязательные поля берут дефолты", func(t *testing.T) {
		// задаём только обязательное, остальное не трогаем — проверяем дефолты
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")

		got, err := Load()
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}

		if got.HTTPAddr != ":8080" {
			t.Errorf("HTTPAddr = %q, want %q", got.HTTPAddr, ":8080")
		}
		if got.ShutdownTimeout != 10*time.Second {
			t.Errorf("ShutdownTimeout = %v, want %v", got.ShutdownTimeout, 10*time.Second)
		}
		if got.LogLevel != slog.LevelInfo {
			t.Errorf("LogLevel = %v, want %v", got.LogLevel, slog.LevelInfo)
		}
	})

	t.Run("значения из окружения переопределяют дефолты", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://x")
		t.Setenv("HTTP_ADDR", ":9000")
		t.Setenv("LOG_LEVEL", "debug")
		t.Setenv("SHUTDOWN_TIMEOUT", "30s")

		got, err := Load()
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}

		if got.HTTPAddr != ":9000" {
			t.Errorf("HTTPAddr = %q, want %q", got.HTTPAddr, ":9000")
		}
		if got.LogLevel != slog.LevelDebug {
			t.Errorf("LogLevel = %v, want %v", got.LogLevel, slog.LevelDebug)
		}
		if got.ShutdownTimeout != 30*time.Second {
			t.Errorf("ShutdownTimeout = %v, want %v", got.ShutdownTimeout, 30*time.Second)
		}
	})

	t.Run("кривой SHUTDOWN_TIMEOUT возвращает ошибку", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://x")
		t.Setenv("SHUTDOWN_TIMEOUT", "abc")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() = nil error, want error про SHUTDOWN_TIMEOUT")
		}
	})
}
