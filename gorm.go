package logpet

import (
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
			"rows_affected": v[5].(string),
		}

		if queryDuration, ok := v[2].(time.Duration); ok && queryDuration >= time.Second {
			gLogger.Logger.SendErrLog(v[3].(string), fields)
		} else {
			gLogger.Logger.SendDebugfLog(v[2].(string), fields)
		}
	case "log":
		gLogger.Logger.SendInfofLog(v[2].(string), nil)
	}
}
