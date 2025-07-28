package alogtest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockTester struct {
	tb       testing.TB
	messages []string
}

func (m *mockTester) Fatalf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)

	m.messages = append(m.messages, msg)
}

func (m *mockTester) Cleanup(cleanup func()) {
	m.tb.Cleanup(cleanup)
}

func (m *mockTester) assert(expectedMessages []string) func() {
	return func() {
		if !slices.Equal(expectedMessages, m.messages) {
			m.tb.Fatalf(
				"messages not equal\n\nexpected:\n[%s]\n\nactual:\n[%s]",
				strings.Join(expectedMessages, "\n"),
				strings.Join(m.messages, "\n"),
			)
		}
	}
}

func newMockTester(tb testing.TB, expectedMessages ...string) *mockTester {
	tester := &mockTester{
		tb: tb,
	}

	tb.Cleanup(tester.assert(expectedMessages))

	return tester
}

func Test_Handler_SuccessCase(t *testing.T) {
	tester := newMockTester(t)

	h := NewHandler(tester,
		(*AssertOptions)(nil),
		Debug("DEBUG MESSAGE",
			"key", "value",
			"integer", 100,
			slog.String("alog.String", "value"),
		),
		Info("INFO MESSAGE"),
		Warn("WARN MESSAGE",
			"owner", "amidman",
			slog.Group("user",
				"age", 100,
			),
		),
		Error("ERROR MESSAGE",
			"owner", "amidman",
			slog.Group("user",
				"sex", "male",
				slog.Group("city",
					"name", "Samara",
				),
			),
		),
		Warn("limit overflow",
			"owner", "amidman",
			slog.Group("user",
				"sex", "male",
				slog.Group("city",
					"budget", "unlimited",
				),
			),
		),
	)

	log := slog.New(h)

	log.Debug("DEBUG MESSAGE",
		"key", "value",
		"integer", 100,
		slog.String("alog.String", "value"),
	)

	log.Info("INFO MESSAGE")

	log = log.With("owner", "amidman")

	log = log.WithGroup("user")

	log.Warn("WARN MESSAGE",
		"age", 100,
	)

	log = log.With("sex", "male")

	log = log.WithGroup("city")

	log.Error("ERROR MESSAGE",
		"name", "Samara",
	)

	log = log.With("budget", "unlimited")

	log.Warn("limit overflow")
}

func Test_Handler_FatalCase_InvalidLength(t *testing.T) {
	const expectedMessage = `
INVALID RECORDS
EXPECTED:
[
----
    time=2023-08-08T20:14:06.000Z level=DEBUG msg="DEBUG MESSAGE" key=value integer=100 alog.String=value
----
    time=2023-08-08T20:14:06.000Z level=DEBUG msg="DEBUG MESSAGE" key=value integer=100 alog.String=value
----
    time=2023-08-08T20:14:06.000Z level=DEBUG msg="DEBUG MESSAGE" key=value integer=100 alog.String=value
----
]
ACTUAL:
[
----
    time=2023-08-08T20:14:06.000Z level=WARN msg="WARN MESSAGE" "MESSAGE\n"="\nMESSAGE" "MESSAGE\n"="\nMESSAGE" "MESSAGE\n"="\nMESSAGE"
----
    time=2023-08-08T20:14:06.000Z level=WARN msg="WARN MESSAGE" "MESSAGE\n"="\nMESSAGE" "MESSAGE\n"="\nMESSAGE" "MESSAGE\n"="\nMESSAGE"
----
]
`
	tester := newMockTester(t, expectedMessage)

	h := NewHandler(tester,
		(*AssertOptions)(nil),
		Debug("DEBUG MESSAGE",
			"key", "value",
			"integer", 100,
			slog.String("alog.String", "value"),
		),
		Debug("DEBUG MESSAGE",
			"key", "value",
			"integer", 100,
			slog.String("alog.String", "value"),
		),
		Debug("DEBUG MESSAGE",
			"key", "value",
			"integer", 100,
			slog.String("alog.String", "value"),
		),
	)

	log := slog.New(h)

	log.Warn("WARN MESSAGE",
		"MESSAGE\n", "\nMESSAGE",
		"MESSAGE\n", "\nMESSAGE",
		"MESSAGE\n", "\nMESSAGE",
	)

	log.Warn("WARN MESSAGE",
		"MESSAGE\n", "\nMESSAGE",
		"MESSAGE\n", "\nMESSAGE",
		"MESSAGE\n", "\nMESSAGE",
	)
}

func Test_Handler_FatalCase_OneNotMatch(t *testing.T) {
	const expectedMessage = `
INVALID RECORD BY 2 INDEX
EXPECTED:
----
    time=2023-08-08T20:14:06.000Z level=DEBUG msg="DEBUG MESSAGE" key=value integer=100 alog.String=value
----
ACTUAL:
----
    time=2023-08-08T20:14:06.000Z level=WARN msg="WARN MESSAGE" "MESSAGE\n"="\nMESSAGE" "MESSAGE\n"="\nMESSAGE" "MESSAGE\n"="\nMESSAGE"
----
`
	tester := newMockTester(t, expectedMessage)

	h := NewHandler(tester,
		(*AssertOptions)(nil),
		Debug("DEBUG MESSAGE",
			"key", "value",
			"integer", 100,
			slog.String("alog.String", "value"),
		),
		Debug("DEBUG MESSAGE",
			"key", "value",
			"integer", 100,
			slog.String("alog.String", "value"),
		),
		Debug("DEBUG MESSAGE",
			"key", "value",
			"integer", 100,
			slog.String("alog.String", "value"),
		),
	)

	log := slog.New(h)

	log.Debug("DEBUG MESSAGE",
		"key", "value",
		"integer", 100,
		slog.String("alog.String", "value"),
	)

	log.Debug("DEBUG MESSAGE",
		"key", "value",
		"integer", 100,
		slog.String("alog.String", "value"),
	)

	log.Warn("WARN MESSAGE",
		"MESSAGE\n", "\nMESSAGE",
		"MESSAGE\n", "\nMESSAGE",
		"MESSAGE\n", "\nMESSAGE",
	)
}

func Test_Handler_SuccessCase_Parallel(t *testing.T) {
	const clientsCount = 1000

	tester := newMockTester(t)

	parallelCase(tester, &AssertOptions{CheckOrder: false}, clientsCount)
}

type fatalfStubTester struct {
	tb    testing.TB
	fatal bool
}

func (f *fatalfStubTester) Cleanup(cleanup func()) {
	f.tb.Cleanup(cleanup)
}

func (f *fatalfStubTester) Fatalf(string, ...any) {
	f.fatal = true
}

func newFatalfStubTester(tb testing.TB) *fatalfStubTester {
	tester := &fatalfStubTester{
		tb: tb,
	}

	tb.Cleanup(func() {
		if !tester.fatal {
			tb.Fatal("expected fatal but nothing happened")
		}
	})

	return tester
}

func Test_Handler_FatalfCase_Parallel(t *testing.T) {
	const clientsCount = 1000

	tester := newFatalfStubTester(t)

	parallelCase(tester, (*AssertOptions)(nil), clientsCount)
}

func parallelCase(tester Tester, opts *AssertOptions, clientsCount int) {
	expectedOperations := []Operation{
		Debug("DEBUG MESSAGE",
			"key", "value",
			"integer", 100,
			slog.String("alog.String", "value"),
		),
		Info("INFO MESSAGE"),
		Warn("WARN MESSAGE",
			"owner", "amidman",
			slog.Group("user",
				"age", 100,
			),
		),
		Error("ERROR MESSAGE",
			"owner", "amidman",
			slog.Group("user",
				"sex", "male",
				slog.Group("city",
					"name", "Samara",
				),
			),
		),
		Warn("limit overflow",
			"owner", "amidman",
			slog.Group("user",
				"sex", "male",
				slog.Group("city",
					"budget", "unlimited",
				),
			),
		),
	}

	h := NewHandler(tester,
		opts,
		multipleOperations(expectedOperations, clientsCount)...,
	)

	logFunc := func() {
		log := slog.New(h)

		log.Debug("DEBUG MESSAGE",
			"key", "value",
			"integer", 100,
			slog.String("alog.String", "value"),
		)

		log.Info("INFO MESSAGE")

		log = log.With("owner", "amidman")

		log = log.WithGroup("user")

		log.Warn("WARN MESSAGE",
			"age", 100,
		)

		log = log.With("sex", "male")

		log = log.WithGroup("city")

		log.Error("ERROR MESSAGE",
			"name", "Samara",
		)

		log = log.With("budget", "unlimited")

		log.Warn("limit overflow")
	}

	wg := sync.WaitGroup{}

	wg.Add(clientsCount)

	for range clientsCount {
		go func() {
			defer wg.Done()

			logFunc()
		}()
	}

	wg.Wait()
}

func multipleOperations(ops []Operation, count int) []Operation {
	result := make([]Operation, 0, count*len(ops))

	for range count {
		result = append(result, ops...)
	}

	return result
}

type errorHandler struct{}

func (errorHandler) WithGroup(string) slog.Handler {
	return nil
}

func (errorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return nil
}

func (errorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (errorHandler) Handle(ctx context.Context, rec slog.Record) error {
	return io.ErrUnexpectedEOF
}

func Test_recordsCollector_Error(t *testing.T) {
	ctx := t.Context()

	recCollector := recordsCollector{
		newHandler: func(io.Writer) slog.Handler {
			return errorHandler{}
		},
		mutates: []func(h slog.Handler) slog.Handler{},
		logs:    &logs{},
	}

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)

	err := recCollector.Handle(ctx, record)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("unexpected error\nexpected:\n%s\nactual:\n%s", io.ErrUnexpectedEOF, err)
	}
}
