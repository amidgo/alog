package alog

import (
	"context"
	"log/slog"
)

type loggerKey struct{}

func Logger(ctx context.Context) *slog.Logger {
	log, ok := ctx.Value(loggerKey{}).(*slog.Logger)
	if !ok {
		return slog.Default()
	}

	return log
}

func Context(ctx context.Context, log *slog.Logger) context.Context {
	if log == nil {
		return ctx
	}

	return context.WithValue(ctx, loggerKey{}, log)
}

func With(ctx context.Context, args ...any) context.Context {
	log := Logger(ctx)

	handler := log.
		Handler().
		WithAttrs(
			argsToAttrSlice(args),
		)

	log = slog.New(handler)

	return Context(ctx, log)
}

func Info(ctx context.Context, msg string, args ...any) {
	Log(ctx, slog.LevelInfo, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	Log(ctx, slog.LevelWarn, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	Log(ctx, slog.LevelError, msg, args...)
}

func Debug(ctx context.Context, msg string, args ...any) {
	Log(ctx, slog.LevelDebug, msg, args...)
}

func Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	log := Logger(ctx)

	log.LogAttrs(ctx, level, msg, argsToAttrSlice(args)...)
}

const badKey = "!BADKEY"

func argsToAttrSlice(args []any) []slog.Attr {
	var (
		attr  slog.Attr
		attrs []slog.Attr
	)

	for len(args) > 0 {
		attr, args = argsToAttr(args)
		attrs = append(attrs, attr)
	}

	return attrs
}

func argsToAttr(args []any) (slog.Attr, []any) {
	switch x := args[0].(type) {
	case string:
		if len(args) == 1 {
			return slog.String(badKey, x), nil
		}

		return slog.Any(x, args[1]), args[2:]

	case slog.Attr:
		return x, args[1:]

	case error:
		return errorAttr(x), args[1:]

	default:
		return slog.Any(badKey, x), args[1:]
	}
}

const ErrorKey = "err"

var nilError = slog.Attr{
	Key:   ErrorKey,
	Value: slog.StringValue("nil"),
}

func errorAttr(err error) slog.Attr {
	if err == nil {
		return nilError
	}

	return slog.String(ErrorKey, err.Error())
}
