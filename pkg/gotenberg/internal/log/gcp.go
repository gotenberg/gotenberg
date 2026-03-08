package log

import "log/slog"

func gcpSeverity(l slog.Level) string {
	switch {
	case l < slog.LevelInfo:
		return "DEBUG"
	case l < slog.LevelWarn:
		return "INFO"
	case l < slog.LevelError:
		return "WARNING"
	case l >= slog.LevelError:
		return "ERROR"
	default:
		return "DEFAULT"
	}
}
