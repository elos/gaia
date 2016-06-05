package services

import (
	"io"
	"log"
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
	l.Logger.Fatal(append([]interface{}{l.prefix}, v...))
}

func (l *logger) Fatalf(format string, v ...interface{}) {
	l.Logger.Fatalf(l.prefix+format, v...)
}

func (l *logger) Print(v ...interface{}) {
	l.Logger.Print(append([]interface{}{l.prefix}, v...))
}

func (l *logger) Printf(format string, v ...interface{}) {
	l.Logger.Printf(l.prefix+format, v...)
}

func (l *logger) WithPrefix(s string) Logger {
	return &logger{
		Logger: l.Logger,
		prefix: l.prefix + s,
	}
}

// --- }}}

// --- TestLogger {{{

type testLogger struct {
	*testing.T
	prefix string
}

func NewTestLogger(t *testing.T) Logger {
	return &testLogger{
		T: t,
	}
}

func (t *testLogger) Print(v ...interface{}) {
	t.T.Log(v...)
}

func (t *testLogger) Printf(format string, v ...interface{}) {
	t.T.Logf(format, v...)
}

func (t *testLogger) Fatal(v ...interface{}) {
	t.T.Fatal(v...)
}

func (t *testLogger) Fatalf(format string, v ...interface{}) {
	t.T.Fatalf(format, v...)
}

func (t *testLogger) WithPrefix(s string) Logger {
	return &testLogger{
		T:      t.T,
		prefix: t.prefix + s,
	}
}

// --- Test Logger }}}
