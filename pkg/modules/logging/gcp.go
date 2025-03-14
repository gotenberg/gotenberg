package logging

import "go.uber.org/zap/zapcore"

func gcpSeverity(l zapcore.Level) string {
	switch l {
	case zapcore.DebugLevel:
		return "DEBUG"
	case zapcore.InfoLevel:
		return "INFO"
	case zapcore.WarnLevel:
		return "WARNING"
	case zapcore.ErrorLevel:
		return "ERROR"
	case zapcore.DPanicLevel:
		return "CRITICAL"
	case zapcore.PanicLevel:
		return "ALERT"
	case zapcore.FatalLevel:
		return "EMERGENCY"
	case zapcore.InvalidLevel:
		return "DEFAULT"
	default:
		return "DEFAULT"
	}
}

func gcpSeverityEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(gcpSeverity(l))
}

func gcpSeverityColorEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	severity := gcpSeverity(l)
	c := levelToColor(l)
	enc.AppendString(c.Add(severity))
}
