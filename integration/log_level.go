package integration

import (
    "errors"
)

// LogLevel is the set of all log levels.
type LogLevel int8

const (
    // CRITICAL is the lowest log level; only errors which will end the program will be propagated.
    CRITICAL LogLevel = iota - 1
    ERROR
    WARNING
    INFO
    DEBUG
    TRACE
)

// Char returns a single-character representation of the log level.
func (l LogLevel) Char() string {
    switch l {
    case CRITICAL:
        return "C"
    case ERROR:
        return "E"
    case WARNING:
        return "W"
    case INFO:
        return "I"
    case DEBUG:
        return "D"
    case TRACE:
        return "T"
    default:
        panic("Unhandled loglevel")
    }
}

// String returns a multi-character representation of the log level.
func (l LogLevel) String() string {
    switch l {
    case CRITICAL:
        return "CRITICAL"
    case ERROR:
        return "ERROR"
    case WARNING:
        return "WARNING"
    case INFO:
        return "INFO"
    case DEBUG:
        return "DEBUG"
    case TRACE:
        return "TRACE"
    default:
        panic("Unhandled loglevel")
    }
}

// Update using the given string value. Fulfills the flag.Value interface.
func (l *LogLevel) Set(s string) error {
    value, err := ParseLogLevel(s)
    if err != nil {
        return err
    }

    *l = value
    return nil
}

// ParseLogLevel translates some potential loglevel strings into their corresponding levels.
func ParseLogLevel(s string) (LogLevel, error) {
    switch s {
    case "CRITICAL", "C":
        return CRITICAL, nil
    case "ERROR", "0", "E":
        return ERROR, nil
    case "WARNING", "1", "W":
        return WARNING, nil
    case "INFO", "2", "I":
        return INFO, nil
    case "DEBUG", "3", "D":
        return DEBUG, nil
    case "TRACE", "4", "T":
        return TRACE, nil
    }
    return CRITICAL, errors.New("couldn't parse log level " + s)
}
