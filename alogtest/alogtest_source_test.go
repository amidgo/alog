package alogtest

import (
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/amidgo/alog"
)

func Test_Source(t *testing.T) {
	handler := NewHandler(t,
		&AssertOptions{
			CheckOrder: true,
			AddSource:  true,
		},
		Debug("debug",
			testSource(49),
		),
		Info("info",
			testSource(50),
		),
		Warn("warn",
			testSource(51),
		),
		Error("error",
			testSource(52),
		),
		Info("start",
			testSource(53),
			slog.String(alog.OpKey, "op"),
		),
		Error("error",
			testSource(54),
			slog.String(alog.OpKey, "op"),
			slog.String(alog.ErrorKey, io.ErrUnexpectedEOF.Error()),
		),
		Info("finish",
			testSource(55),
			slog.String(alog.OpKey, "op"),
		),
	)

	ctx := alog.Context(t.Context(), handler)

	alog.Debug(ctx, "debug")
	alog.Info(ctx, "info")
	alog.Warn(ctx, "warn")
	alog.Error(ctx, "error")
	op := alog.Start(ctx, "op")
	op.Error(io.ErrUnexpectedEOF)
	op.Finish()
}

func testSource(line int) slog.Attr {
	return slog.Any(
		slog.SourceKey,
		&slog.Source{
			Function: "Test_Source",
			File:     getCurrentFilePath(),
			Line:     line,
		},
	)
}

func getCurrentFilePath() string {
	pcs := make([]uintptr, 1)

	// Пропускаем:
	//  0 - текущий вызов runtime.Callers
	//  1 - сама функция getCurrentFilePath
	skip := 2

	// Получаем Program Counters (PC) для текущего кадра
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return "unknown"
	}

	// Преобразуем PC в Frames
	frames := runtime.CallersFrames(pcs[:n])
	frame, _ := frames.Next()

	// Очищаем путь от лишних частей (опционально)
	cleanPath := frame.File
	if idx := strings.LastIndex(cleanPath, "/vendor/"); idx != -1 {
		cleanPath = cleanPath[idx+len("/vendor/"):]
	}

	// Получаем абсолютный путь
	absPath, _ := filepath.Abs(cleanPath)
	return absPath
}
