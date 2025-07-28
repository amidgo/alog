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

	"github.com/amidgo/alog/alogtest"
)

func Test_Context_Handler(t *testing.T) {
	handler := slog.NewJSONHandler(io.Discard, nil)

	ctx := Context(t.Context(), handler)

	handlerFromCtx := Handler(ctx)

	if handlerFromCtx != handler {
		t.Fatal("handler not equal handler from ctx")
	}

	if ctx == t.Context() {
		t.Fatal("unexpected non modify retuned context")
	}
}

func Test_Context_nil_slog_Handler(t *testing.T) {
	ctx := t.Context()

	alogCtx := Context(ctx, nil)

	if ctx != alogCtx {
		t.Fatal("context has been modified")
	}
}

func Test_Handler_DefaultHandler(t *testing.T) {
	ctx := t.Context()

	handlerFromCtx := Handler(ctx)

	if handlerFromCtx != slog.Default().Handler() {
		t.Fatal("return non default handler")
	}
}

func Test_withSource(t *testing.T) {
	out := new(bytes.Buffer)

	h := slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})

	ctx := Context(t.Context(), h)

	Info(ctx, "Hello World!")

	t.Logf("out: %q", out.String())
}

func Test_argsToAttrSlice(t *testing.T) {
	type argsToAttrSliceTest struct {
		name          string
		args          []any
		expectedAttrs []slog.Attr
	}

	ctx := t.Context()

	tests := []argsToAttrSliceTest{
		{
			name: "string single args bad key",
			args: []any{
				"key",
			},
			expectedAttrs: []slog.Attr{
				{
					Key:   "!BADKEY",
					Value: slog.StringValue("key"),
				},
			},
		},
		{
			name: "string many args bad key",
			args: []any{
				"key",
				"value",
				100,
			},
			expectedAttrs: []slog.Attr{
				slog.String("key", "value"),
				slog.Any("!BADKEY", 100),
			},
		},
		{
			name: "error attr convert",
			args: []any{
				"key", "value",
				http.ErrServerClosed,
				"int", 100,
				"err", bufio.ErrTooLong,
				slog.Any("ctx", ctx),
				io.ErrUnexpectedEOF,
			},
			expectedAttrs: []slog.Attr{
				slog.String("key", "value"),
				slog.String(ErrorKey, http.ErrServerClosed.Error()),
				slog.Int("int", 100),
				slog.Any("err", bufio.ErrTooLong),
				slog.Any("ctx", ctx),
				slog.String(ErrorKey, io.ErrUnexpectedEOF.Error()),
			},
		},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			attrs := argsToAttrSlice(tst.args)

			if len(attrs) != len(tst.expectedAttrs) {
				t.Fatalf("args not equal\n\nexpected:\n%+v\n\nactual:\n%+v", tst.expectedAttrs, attrs)
			}

			for i := range attrs {
				if !reflect.DeepEqual(attrs[i], tst.expectedAttrs[i]) {
					t.Fatalf("compare %d arg, args not equal\n\nexpected:\n%+v\n\nactual:\n%+v", i, tst.expectedAttrs[i], attrs[i])
				}
			}
		})
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
		Level:       slog.LevelError,
		ReplaceAttr: replaceTimeKey(time.Now()),
	}

	alogBuf := &bytes.Buffer{}
	alogHandler := slog.NewTextHandler(alogBuf, opts)
	ctx := Context(t.Context(), alogHandler)

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
		Level:       slog.LevelDebug,
		ReplaceAttr: replaceTimeKey(time.Now()),
	}

	alogBuf := &bytes.Buffer{}
	alogHandler := slog.NewTextHandler(alogBuf, opts)
	ctx := Context(t.Context(), alogHandler)

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
	ctx = WithAttrs(ctx, argsToAttrSlice(alogArgs)...)
	slogLogger = slog.New(slogLogger.Handler().WithAttrs(argsToAttrSlice(slogArgs)))

	Log(ctx, slog.LevelDebug, "debug message", alogArgs...)
	slogLogger.Log(ctx, slog.LevelDebug, "debug message", slogArgs...)

	LogAttrs(ctx, slog.LevelDebug, "debug log attrs", argsToAttrSlice(alogArgs)...)
	slogLogger.LogAttrs(ctx, slog.LevelDebug, "debug log attrs", argsToAttrSlice(slogArgs)...)

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

func Test_Start(t *testing.T) {
	const opName = "Test_Start"

	handler := alogtest.NewHandler(t,
		(*alogtest.AssertOptions)(nil),
		alogtest.Info("start",
			OpKey, opName,
			"key", "value",
			"age", 10,
		),
		alogtest.Error("error",
			OpKey, opName,
			"key", "value",
			"age", 10,
			ErrorKey, io.ErrUnexpectedEOF.Error(),
			"reason", "db",
		),
		alogtest.Info("finish",
			OpKey, opName,
			"key", "value",
			"age", 10,
		),
	)

	ctx := Context(t.Context(), handler)

	op := Start(ctx,
		opName,
		"key", "value",
		"age", 10,
	)
	defer op.Finish()

	op.Error(io.ErrUnexpectedEOF, "reason", "db")
}

func Test_WithGroup(t *testing.T) {
	handler := alogtest.NewHandler(t,
		(*alogtest.AssertOptions)(nil),
		alogtest.Info("initial"),
		alogtest.Info("get update", slog.Group("user", "age", 10, slog.Group("money", "value", 0))),
	)

	ctx := Context(t.Context(), handler)

	Info(ctx, "initial")

	ctx = WithGroup(ctx, "user")
	ctx = WithAttrs(ctx, slog.Int("age", 10))
	ctx = WithGroup(ctx, "money")
	ctx = WithAttrs(ctx, slog.Int("value", 0))

	Info(ctx, "get update")
}
