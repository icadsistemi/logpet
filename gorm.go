package logpet

import (
	"strconv"
	"time"
)

func CreateGormLogger(logger *StandardLogger) *GormLogger {
	return &GormLogger{Logger: logger}
}

type GormLogger struct {
	Logger *StandardLogger
}

// Print handles log events from Gorm for the custom logger.
func (gLogger *GormLogger) Print(v ...interface{}) {

	switch v[0] {
	case "sql":
		fields := map[string]string{
			"section":       "database",
			"query_time":    v[2].(time.Duration).String(),
			"function_line": v[1].(string),
			"rows_affected": strconv.FormatInt(v[5].(int64), 10),
		}

		gLogger.Logger.SendDebugfLog(v[3].(string), fields)
	case "log":
		gLogger.Logger.SendInfofLog(v[3].(string), nil)
	}
}
