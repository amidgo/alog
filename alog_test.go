package alog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"testing"
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
	alogBuf := &bytes.Buffer{}
	alogLogger := slog.New(slog.NewTextHandler(alogBuf, nil))
	ctx := Context(t.Context(), alogLogger)

	slogBuf := &bytes.Buffer{}
	slogLogger := slog.New(slog.NewTextHandler(slogBuf, nil))

	slogArgs := []any{
		"key", "value",
		errorAttr(http.ErrServerClosed),
		"int", 100,
		slog.Any("ctx", ctx),
		errorAttr(io.ErrUnexpectedEOF),
	}
	alogArgs := []any{
		"key", "value",
		http.ErrServerClosed,
		"int", 100,
		slog.Any("ctx", ctx),
		io.ErrUnexpectedEOF,
	}

	fullLogScenario(ctx, slogLogger, alogArgs, slogArgs)

	if alogBuf.String() != slogBuf.String() {
		t.Fatalf("buffers not equal, alogBuf: %s, slogBuf: %s", alogBuf, slogBuf)
	}
}

func Test_Log_Methods(t *testing.T) {
	alogBuf := &bytes.Buffer{}
	alogLogger := slog.New(slog.NewTextHandler(alogBuf, nil))
	ctx := Context(t.Context(), alogLogger)

	slogBuf := &bytes.Buffer{}
	slogLogger := slog.New(slog.NewTextHandler(slogBuf, nil))

	args := []any{
		"key", "value",
		"int", 100,
		slog.Any("ctx", ctx),
		errorAttr(io.ErrUnexpectedEOF),
	}

	fullLogScenario(ctx, slogLogger, args, args)

	if alogBuf.String() != slogBuf.String() {
		t.Fatalf("buffers not equal, alogBuf: %s, slogBuf: %s", alogBuf, slogBuf)
	}
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
