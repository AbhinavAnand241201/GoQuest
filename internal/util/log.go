package util

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
	logMutex    sync.Mutex
	verbose     bool
)

func init() {
	infoLogger = log.New(os.Stdout, "", 0)
	errorLogger = log.New(os.Stderr, "", 0)
}

// SetVerbose enables or disables verbose logging.
func SetVerbose(enabled bool) {
	logMutex.Lock()
	verbose = enabled
	logMutex.Unlock()
}

// SetOutput sets the output destination for both info and error logs.
// This is primarily used for testing.
func SetOutput(w io.Writer) {
	logMutex.Lock()
	defer logMutex.Unlock()
	if w == nil {
		infoLogger = log.New(os.Stdout, "", 0)
		errorLogger = log.New(os.Stderr, "", 0)
		return
	}
	infoLogger = log.New(w, "", 0)
	errorLogger = log.New(w, "", 0)
}

// LogInfo logs an informational message with a timestamp.
func LogInfo(format string, v ...interface{}) {
	logMutex.Lock()
	defer logMutex.Unlock()
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, v...)
	infoLogger.Printf("%s INFO: %s", timestamp, msg)
}

// LogDebug logs a debug message with a timestamp if verbose mode is enabled.
func LogDebug(format string, v ...interface{}) {
	logMutex.Lock()
	defer logMutex.Unlock()
	if !verbose {
		return
	}
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, v...)
	infoLogger.Printf("%s DEBUG: %s", timestamp, msg)
}

// LogError logs an error message with a timestamp.
func LogError(format string, v ...interface{}) {
	logMutex.Lock()
	defer logMutex.Unlock()
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, v...)
	errorLogger.Printf("%s ERROR: %s", timestamp, msg)
}

// LogFatal logs a fatal error message and exits with status code 1.
func LogFatal(format string, v ...interface{}) {
	logMutex.Lock()
	defer logMutex.Unlock()
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, v...)
	errorLogger.Printf("%s FATAL: %s", timestamp, msg)
	osExit(1)
}
