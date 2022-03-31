package logging

import (
	"os"

	log "github.com/sirupsen/logrus"
	"go.elastic.co/ecslogrus"
)

func LogErrors(filename string) (*os.File, error) {
	log.SetFormatter(&log.JSONFormatter{})
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	log.SetOutput(f)

	return f, nil
}

func NewLogger() *log.Logger {
	logger := log.New()
	logger.SetLevel(log.DebugLevel)
	logger.SetFormatter(&ecslogrus.Formatter{})
	return logger
}
