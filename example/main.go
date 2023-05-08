package main

import (
	"github.com/google/uuid"
	"time"

	logger "github.com/imkiptoo/go-logger"
)

func main() {
	log, err := logger.New("example", "logs", "database", "config.yaml")
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
