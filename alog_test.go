package alog

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func Test_Context_Logger(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	ctx := Context(t.Context(), logger)

	loggerFromCtx := Logger(ctx)

	if logger != loggerFromCtx {
		t.Fatal("logger not equal logger from ctx")
	}
}

func Test_errorAttr(t *testing.T) {
	type errorAttrTest struct {
		Name         string
		Error        error
		ExpectedAttr slog.Attr
	}

	tests := []errorAttrTest{
		{
			Name:         "nil error",
			Error:        nil,
			ExpectedAttr: slog.String(ErrorKey, "nil"),
		},
		{
			Name:         "context.Canceled error",
			Error:        context.Canceled,
			ExpectedAttr: slog.String(ErrorKey, "context canceled"),
		},
		{
			Name:         "wrapped error",
			Error:        fmt.Errorf("failed to do, %w", context.Canceled),
			ExpectedAttr: slog.String(ErrorKey, "failed to do, context canceled"),
		},
	}

	for _, tst := range tests {
		t.Run(tst.Name,
			func(t *testing.T) {
				attr := errorAttr(tst.Error)

				if !reflect.DeepEqual(attr, tst.ExpectedAttr) {
					t.Fatalf("attrs not equal, expected %+v, actual %+v", tst.ExpectedAttr, attr)
				}
			},
		)
	}
}

func Test_Error(t *testing.T) {
	opts := &slog.HandlerOptions{
		ReplaceAttr: replaceTimeKey(time.Now()),
	}

	alogBuf := &bytes.Buffer{}
	alogLogger := slog.New(slog.NewTextHandler(alogBuf, opts))
	ctx := Context(t.Context(), alogLogger)

	slogBuf := &bytes.Buffer{}
	slogLogger := slog.New(slog.NewTextHandler(slogBuf, opts))

	slogArgs := []any{
		"key", "value",
		errorAttr(http.ErrServerClosed),
		"int", 100,
		"err", bufio.ErrTooLong,
		slog.Any("ctx", ctx),
		errorAttr(io.ErrUnexpectedEOF),
	}
	alogArgs := []any{
		"key", "value",
		http.ErrServerClosed,
		"int", 100,
		"err", bufio.ErrTooLong,
		slog.Any("ctx", ctx),
		io.ErrUnexpectedEOF,
	}

	fullLogScenario(ctx, slogLogger, alogArgs, slogArgs)
	compareBuffers(t, slogBuf, alogBuf)
}

func Test_Log_Methods(t *testing.T) {
	opts := &slog.HandlerOptions{
		ReplaceAttr: replaceTimeKey(time.Now()),
	}

	alogBuf := &bytes.Buffer{}
	alogLogger := slog.New(slog.NewTextHandler(alogBuf, opts))
	ctx := Context(t.Context(), alogLogger)

	slogBuf := &bytes.Buffer{}
	slogLogger := slog.New(slog.NewTextHandler(slogBuf, opts))

	args := []any{
		"key", "value",
		"int", 100,
		slog.Any("ctx", ctx),
		errorAttr(io.ErrUnexpectedEOF),
	}

	fullLogScenario(ctx, slogLogger, args, args)
	compareBuffers(t, slogBuf, alogBuf)
}

func fullLogScenario(ctx context.Context, slogLogger *slog.Logger, alogArgs []any, slogArgs []any) {
	Log(ctx, slog.LevelDebug, "debug message", alogArgs...)
	slogLogger.Log(ctx, slog.LevelDebug, "debug message", slogArgs...)

	Debug(ctx, "debug message", alogArgs...)
	slogLogger.DebugContext(ctx, "debug message", slogArgs...)

	Info(ctx, "information", alogArgs...)
	slogLogger.InfoContext(ctx, "information", slogArgs...)

	Warn(ctx, "warning", alogArgs...)
	slogLogger.WarnContext(ctx, "warning", slogArgs...)

	Error(ctx, "error", alogArgs...)
	slogLogger.ErrorContext(ctx, "error", slogArgs...)

	slogLogger = slogLogger.With("added", "attr")
	ctx = With(ctx, "added", "attr")

	Log(ctx, slog.LevelDebug, "debug message", alogArgs...)
	slogLogger.Log(ctx, slog.LevelDebug, "debug message", slogArgs...)

	Debug(ctx, "debug message", alogArgs...)
	slogLogger.DebugContext(ctx, "debug message", slogArgs...)

	Info(ctx, "information", alogArgs...)
	slogLogger.InfoContext(ctx, "information", slogArgs...)

	Warn(ctx, "warning", alogArgs...)
	slogLogger.WarnContext(ctx, "warning", slogArgs...)

	Error(ctx, "error", alogArgs...)
	slogLogger.ErrorContext(ctx, "error", slogArgs...)
}

func compareBuffers(t *testing.T, slogBuf, alogBuf fmt.Stringer) {
	if alogBuf.String() != slogBuf.String() {
		t.Fatalf("buffers not equal,\nalogBuf:\n%s\nslogBuf:\n%s", alogBuf, slogBuf)
	}
}

func replaceTimeKey(staticTime time.Time) func(_ []string, a slog.Attr) slog.Attr {
	return func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Time(slog.TimeKey, staticTime)
		}

		return a
	}
}
