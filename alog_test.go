package alog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
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

func Test_Error(t *testing.T) {
	type errorTest struct {
		Name         string
		Error        error
		ExpectedAttr slog.Attr
	}

	tests := []errorTest{
		{
			Name:         "nil error",
			Error:        nil,
			ExpectedAttr: slog.String("err", "nil"),
		},
		{
			Name:         "context.Canceled error",
			Error:        context.Canceled,
			ExpectedAttr: slog.String("err", "context canceled"),
		},
		{
			Name:         "wrapped error",
			Error:        fmt.Errorf("failed to do, %w", context.Canceled),
			ExpectedAttr: slog.String("err", "failed to do, context canceled"),
		},
	}

	for _, tst := range tests {
		t.Run(tst.Name,
			func(t *testing.T) {
				attr := Error(tst.Error)

				if !reflect.DeepEqual(attr, tst.ExpectedAttr) {
					t.Fatalf("attrs not equal, expected %+v, actual %+v", tst.ExpectedAttr, attr)
				}
			},
		)
	}
}

func Test_Log_With(t *testing.T) {
	alogBuf := &bytes.Buffer{}
	alogLogger := slog.New(slog.NewTextHandler(alogBuf, nil))
	ctx := Context(t.Context(), alogLogger)

	slogBuf := &bytes.Buffer{}
	slogLogger := slog.New(slog.NewTextHandler(slogBuf, nil))

	args := []any{
		"key", "value",
		"int", 100,
		slog.Any("ctx", ctx),
		Error(io.ErrUnexpectedEOF),
	}

	slogLogger = slogLogger.With("added", "attr")
	ctx = With(ctx, "added", "attr")

	Log(ctx, slog.LevelDebug, "debug message", args...)
	slogLogger.Log(ctx, slog.LevelDebug, "debug message", args...)

	Log(ctx, slog.LevelInfo, "information", args...)
	slogLogger.Log(ctx, slog.LevelInfo, "information", args...)

	Log(ctx, slog.LevelWarn, "warning", args...)
	slogLogger.Log(ctx, slog.LevelWarn, "warning", args...)

	Log(ctx, slog.LevelError, "error", args...)
	slogLogger.Log(ctx, slog.LevelError, "error", args...)

	if alogBuf.String() != slogBuf.String() {
		t.Fatalf("buffers not equal, alogBuf: %s, slogBuf: %s", alogBuf, slogBuf)
	}
}
