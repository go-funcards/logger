package logger

import (
	"github.com/sirupsen/logrus"
	golog "log"
	"os"
	"time"
)

const EnvLogLevel = "LOG_LEVEL"

var log *logrus.Logger

func init() {
	level := os.Getenv(EnvLogLevel)
	if level == "" {
		level = "info"
	}

	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		golog.Fatal(err)
	}

	log = logrus.New()
	log.Level = lvl
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
		PrettyPrint:     lvl >= logrus.DebugLevel,
	}
	log.Out = os.Stdout
}

func GetLog() *logrus.Logger {
	return log
}
