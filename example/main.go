package main

import (
	"github.com/google/uuid"
	"time"

	"github.com/imkiptoo/logger"
)

func main() {
	log, err := logger.New("example", "logs", "database", logger.Config{
		Level:     "debug",
		Frequency: "hourly",
		MaxSize:   "16kb",
		Console:   true,
		Compress:  true,
	})
	if err != nil {
		panic(err)
	}

	for {
		go log.Debugf(uuid.New().String())
		go log.Infof(uuid.New().String())
		go log.Warningf(uuid.New().String())
		go log.Jedif(uuid.New().String())
		go log.Errorf(uuid.New().String())
		time.Sleep(1 * time.Millisecond)
	}
}
