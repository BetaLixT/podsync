package podsync

import "time"

type LogLevel string

const (
	LogInf LogLevel = "inf"
	LogWrn LogLevel = "wrn"
	LogErr LogLevel = "err"
)

type Log struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	Args      []any
}

type Logger interface {
	Inf(msg string, args ...any)
	Wrn(msg string, args ...any)
	Err(msg string, args ...any)
	GetAll() []Log
}

// should be internal

type logger struct {
	logs []Log
}

func NewLogger() *logger {
	return &logger{[]Log{}}
}

func (l *logger) log(level LogLevel, msg string, args ...any) {
	l.logs = append(l.logs, Log{
		time.Now(), // timestamp
		level,      // level
		msg,        // msg
		args,       // args
	})
}

func (l *logger) Inf(msg string, args ...any) {
	l.log(LogInf, msg, args...)
}

func (l *logger) Wrn(msg string, args ...any) {
	l.log(LogWrn, msg, args...)
}

func (l *logger) Err(msg string, args ...any) {
	l.log(LogErr, msg, args...)
}

func (l *logger) GetAll() []Log {
	return l.logs
}
