package asynctask

import (
	"log"
	"os"
)

var (
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
)

func init() {
	infoLogger = log.New(os.Stdout, "asynctask: INFO: ", log.Ldate|log.Ltime)
	warnLogger = log.New(os.Stdout, "asynctask: WARNING: ", log.Ldate|log.Ltime)
	errorLogger = log.New(os.Stderr, "asynctask: ERROR: ", log.Ldate|log.Ltime)
}
