package logging

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// New configures a JSON slog logger for stdout.
func New(levelText string) (*slog.Logger, error) {
	level, err := parseLevel(levelText)
	if err != nil {
		return nil, err
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler), nil
}

func parseLevel(levelText string) (slog.Level, error) {
	if levelText == "" {
		return slog.LevelInfo, nil
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToLower(levelText))); err != nil {
		return 0, fmt.Errorf("invalid LOG_LEVEL %q: %w", levelText, err)
	}

	return level, nil
}
