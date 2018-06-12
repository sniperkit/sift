package sift

import (
	"log"
	"os"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stderr, "Error: ", 0)
}

func Logger(logger *log.Logger) {
	logger = logger
}
