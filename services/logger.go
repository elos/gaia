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
}

func NewLogger(out io.Writer) *log.Logger {
	return log.New(out, "", log.Ldate|log.Ltime|log.Lshortfile)
}

func NewTestLogger(t *testing.T) Logger {
	return &testLogger{
		T: t,
	}
}

type testLogger struct {
	*testing.T
}

func (t *testLogger) Print(v ...interface{}) {
	t.T.Log(v...)
}

func (t *testLogger) Printf(format string, v ...interface{}) {
	t.T.Logf(format, v...)
}
