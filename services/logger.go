package services

import (
	"fmt"
	"io"
	"log"
	"os"
	"testing"
)

type Logger interface {
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Print(v ...interface{})
	Printf(format string, v ...interface{})

	WithPrefix(s string) Logger
}

// --- Logger {{{

type logger struct {
	*log.Logger
	prefix string
}

func NewLogger(out io.Writer) Logger {
	return &logger{
		Logger: log.New(out, "", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (l *logger) Fatal(v ...interface{}) {
	l.Logger.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

func (l *logger) Fatalf(format string, v ...interface{}) {
	l.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

func (l *logger) Print(v ...interface{}) { l.Output(2, fmt.Sprint(v...)) }

func (l *logger) Printf(format string, v ...interface{}) { l.Output(2, fmt.Sprintf(format, v...)) }

func (l *logger) WithPrefix(s string) Logger {
	return &logger{
		Logger: l.Logger,
		prefix: l.prefix + s,
	}
}

// --- }}}

// --- TestLogger {{{

type testLogger struct {
	testing.TB
	prefix string
}

func NewTestLogger(tb testing.TB) Logger {
	return &testLogger{
		TB: tb,
	}
}

func (t *testLogger) Print(v ...interface{}) {
	t.TB.Log(v...)
}

func (t *testLogger) Printf(format string, v ...interface{}) {
	t.TB.Logf(format, v...)
}

func (t *testLogger) Fatal(v ...interface{}) {
	t.TB.Fatal(v...)
}

func (t *testLogger) Fatalf(format string, v ...interface{}) {
	t.TB.Fatalf(format, v...)
}

func (t *testLogger) WithPrefix(s string) Logger {
	return &testLogger{
		TB:     t.TB,
		prefix: t.prefix + s,
	}
}

// --- Test Logger }}}
