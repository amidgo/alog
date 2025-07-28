package alogtest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"slices"
	"strings"
	"sync"
	"time"
)

func Time() slog.Attr {
	return slog.Time(
		slog.TimeKey,
		// date of release go 1.21, see https://github.com/golang/go/commit/c19c4c566c63818dfd059b352e52c4710eecf14d
		time.Date(2023, time.August, 8, 20, 14, 6, 0, time.UTC),
	)
}

type Operation struct {
	Level slog.Level
	Msg   string
	Args  []any
}

func Debug(msg string, args ...any) Operation {
	return Operation{
		Level: slog.LevelDebug,
		Msg:   msg,
		Args:  args,
	}
}

func Info(msg string, args ...any) Operation {
	return Operation{
		Level: slog.LevelInfo,
		Msg:   msg,
		Args:  args,
	}
}

func Warn(msg string, args ...any) Operation {
	return Operation{
		Level: slog.LevelWarn,
		Msg:   msg,
		Args:  args,
	}
}

func Error(msg string, args ...any) Operation {
	return Operation{
		Level: slog.LevelError,
		Msg:   msg,
		Args:  args,
	}
}

type Tester interface {
	Fatalf(string, ...any)
	Cleanup(func())
}

type AssertOptions struct {
	CheckOrder bool
	AddSource  bool
}

func assertOptionsCheckOrder(opts *AssertOptions) bool {
	if opts != nil {
		return opts.CheckOrder
	}

	return false
}

func assertOptionsAddSource(opts *AssertOptions) bool {
	if opts != nil {
		return opts.AddSource
	}

	return false
}

func NewHandler(
	tester Tester,
	opts *AssertOptions,
	ops ...Operation,
) slog.Handler {
	newHandler := newTextHandler(false)

	h := recordsCollector{
		newHandler: newHandler,
		mutates:    []func(h slog.Handler) slog.Handler{},
		logs:       &logs{},
	}

	assert := assertOperationsExecuted(tester, h.logs, opts, ops)

	tester.Cleanup(assert)

	return h
}

func assertOperationsExecuted(tester Tester, logs *logs, opts *AssertOptions, ops []Operation) func() {
	return func() {
		actualRecords := logs.Records()

		newHandler := newTextHandler(assertOptionsAddSource(opts))

		expectedRecords := makeExpectedRecords(newHandler, ops)

		if !assertOptionsCheckOrder(opts) {
			slices.Sort(actualRecords)
			slices.Sort(expectedRecords)
		}

		if len(expectedRecords) != len(actualRecords) {
			fatalfInvalidRecords(tester, expectedRecords, actualRecords)

			return
		}

		for i := range expectedRecords {
			expectedRecord := expectedRecords[i]
			actualRecord := actualRecords[i]

			if expectedRecord != actualRecord {
				fatalfInvalidRecordByIndex(tester, i, expectedRecord, actualRecord)

				return
			}
		}
	}
}

func makeExpectedRecords(newHandler func(io.Writer) slog.Handler, ops []Operation) []string {
	ctx := context.Background()
	expectedRecords := make([]string, 0)

	buf := new(bytes.Buffer)

	for _, op := range ops {
		buf.Reset()

		h := newHandler(buf)
		log := slog.New(h)

		log.Log(ctx, op.Level, op.Msg, op.Args...)

		expectedRecords = append(expectedRecords, buf.String())
	}

	return expectedRecords
}

func fatalfInvalidRecords(tester Tester, expectedRecords, actualRecords []string) {
	tester.Fatalf("\nINVALID RECORDS\nEXPECTED:\n%s\nACTUAL:\n%s\n",
		recordsForTesterMessage(expectedRecords),
		recordsForTesterMessage(actualRecords),
	)
}

func recordsForTesterMessage(records []string) string {
	const sep = "----"

	bld := new(strings.Builder)

	bld.WriteByte('[')
	bld.WriteByte('\n')

	for _, rec := range records {
		bld.WriteString(sep)
		bld.WriteByte('\n')
		bld.WriteString("    ")
		bld.WriteString(rec)
	}

	bld.WriteString(sep)
	bld.WriteByte('\n')

	bld.WriteByte(']')

	return bld.String()
}

func fatalfInvalidRecordByIndex(tester Tester, index int, expectedRecord, actualRecord string) {
	tester.Fatalf(
		"\nINVALID RECORD BY %d INDEX\nEXPECTED:\n%s\nACTUAL:\n%s\n",
		index,
		recordForTesterMessage(expectedRecord),
		recordForTesterMessage(actualRecord),
	)
}

func recordForTesterMessage(record string) string {
	const sep = "----"

	bld := new(strings.Builder)

	bld.WriteString(sep)
	bld.WriteByte('\n')
	bld.WriteString("    ")
	bld.WriteString(record)
	bld.WriteString(sep)

	return bld.String()
}

type recordsCollector struct {
	newHandler func(io.Writer) slog.Handler
	mutates    []func(h slog.Handler) slog.Handler
	logs       *logs
}

var _ slog.Handler = (*recordsCollector)(nil)

func (h recordsCollector) clone() recordsCollector {
	h.mutates = slices.Clip(h.mutates)

	return h
}

func (h recordsCollector) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h recordsCollector) WithAttrs(attrs []slog.Attr) slog.Handler {
	h = h.clone()

	h.mutates = append(h.mutates, withAttrs(attrs))

	return h
}

func withAttrs(attrs []slog.Attr) func(slog.Handler) slog.Handler {
	return func(h slog.Handler) slog.Handler {
		return h.WithAttrs(attrs)
	}
}

func (h recordsCollector) WithGroup(groupName string) slog.Handler {
	h = h.clone()

	h.mutates = append(h.mutates, withGroup(groupName))

	return h
}

func withGroup(groupName string) func(slog.Handler) slog.Handler {
	return func(h slog.Handler) slog.Handler {
		return h.WithGroup(groupName)
	}
}

func (h recordsCollector) Handle(ctx context.Context, record slog.Record) error {
	buf := new(bytes.Buffer)

	handler := h.newHandler(buf)
	for _, mutate := range h.mutates {
		handler = mutate(handler)
	}

	err := handler.Handle(ctx, record)
	if err != nil {
		return fmt.Errorf("handler.Handle: %w", err)
	}

	h.logs.push(buf.String())

	return nil
}

type logs struct {
	mu      sync.Mutex
	records []string
}

func (l *logs) Records() []string {
	records := make([]string, len(l.records))

	l.mu.Lock()
	copy(records, l.records)
	l.mu.Unlock()

	return records
}

func (l *logs) push(record string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.records = append(l.records, record)
}

const minLevel slog.Level = math.MinInt

func newTextHandler(addSource bool) func(io.Writer) slog.Handler {
	return func(w io.Writer) slog.Handler {
		return slog.NewTextHandler(w,
			&slog.HandlerOptions{
				AddSource:   addSource,
				Level:       minLevel,
				ReplaceAttr: replaceTimeAttr,
			},
		)
	}
}

func replaceTimeAttr(_ []string, attr slog.Attr) slog.Attr {
	if attr.Key == slog.TimeKey {
		return Time()
	}

	return attr
}
