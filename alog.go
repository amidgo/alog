package alog

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

type handlerKey struct{}

func Handler(ctx context.Context) slog.Handler {
	handler, ok := ctx.Value(handlerKey{}).(slog.Handler)
	if !ok {
		return slog.Default().Handler()
	}

	return handler
}

func Context(ctx context.Context, h slog.Handler) context.Context {
	if h == nil {
		return ctx
	}

	return context.WithValue(ctx, handlerKey{}, h)
}

func With(ctx context.Context, args ...any) context.Context {
	h := Handler(ctx)

	h = h.WithAttrs(
		argsToAttrSlice(args),
	)

	return Context(ctx, h)
}

func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	h := Handler(ctx)

	h = h.WithAttrs(attrs)

	return Context(ctx, h)
}

func WithGroup(ctx context.Context, groupName string) context.Context {
	h := Handler(ctx)

	h = h.WithGroup(groupName)

	return Context(ctx, h)
}

func Info(ctx context.Context, msg string, args ...any) {
	alog(ctx, slog.LevelInfo, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	alog(ctx, slog.LevelWarn, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	alog(ctx, slog.LevelError, msg, args...)
}

func Debug(ctx context.Context, msg string, args ...any) {
	alog(ctx, slog.LevelDebug, msg, args...)
}

func Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	alog(ctx, level, msg, args...)
}

func LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	alogAttrs(ctx, level, msg, attrs...)
}

func alog(ctx context.Context, level slog.Level, msg string, args ...any) {
	h := Handler(ctx)
	if !h.Enabled(ctx, level) {
		return
	}

	pcs := [1]uintptr{}
	runtime.Callers(3, pcs[:])

	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	attrs := argsToAttrSlice(args)

	r.AddAttrs(attrs...)

	_ = h.Handle(ctx, r)
}

func alogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	h := Handler(ctx)
	if !h.Enabled(ctx, level) {
		return
	}

	pcs := [1]uintptr{}
	runtime.Callers(3, pcs[:])

	r := slog.NewRecord(time.Now(), level, msg, pcs[0])

	r.AddAttrs(attrs...)

	_ = h.Handle(ctx, r)
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

const OpKey = "op"

type Operation struct {
	ctx context.Context
}

func Start(ctx context.Context, opName string, additionalArgs ...any) Operation {
	const minArgsAmount = 1

	args := make([]any, 0, len(additionalArgs)+minArgsAmount)

	args = append(args, slog.String(OpKey, opName))
	args = append(args, additionalArgs...)

	ctx = With(ctx, args...)

	op := Operation{
		ctx: ctx,
	}

	alog(op.ctx, slog.LevelInfo, "start")

	return op
}

func (op Operation) Finish() {
	alogAttrs(op.ctx, slog.LevelInfo, "finish")
}

func (op Operation) Error(err error, additionalArgs ...any) {
	const minArgsAmount = 1

	args := make([]any, 0, len(additionalArgs)+minArgsAmount)

	args = append(args, errorAttr(err))
	args = append(args, additionalArgs...)

	alog(op.ctx, slog.LevelError, "error", args...)
}
