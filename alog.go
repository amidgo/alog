package alog

import (
	"context"
	"log/slog"
)

type loggerKey struct{}

func Logger(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggerKey{}).(*slog.Logger)
	if !ok {
		return slog.Default()
	}

	return logger
}

func Context(ctx context.Context, logger *slog.Logger) context.Context {
	if logger == nil {
		return ctx
	}

	return context.WithValue(ctx, loggerKey{}, logger)
}

func With(ctx context.Context, args ...any) context.Context {
	logger := Logger(ctx)

	logger = logger.With(args...)

	return Context(ctx, logger)
}

func Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	logger := Logger(ctx)

	logger.Log(ctx, level, msg, args...)
}

const ErrorKey = "err"

var nilError = slog.Attr{
	Key:   ErrorKey,
	Value: slog.StringValue("nil"),
}

func Error(err error) slog.Attr {
	if err == nil {
		return nilError
	}

	return slog.String(ErrorKey, err.Error())
}
