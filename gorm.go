package logpet

import (
	"fmt"
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

		queryfields := map[string]string{
			"execution_time": v[2].(time.Duration).String(),
			"arguments":      fmt.Sprintf("%v", v[4]),
			"rows_affected":  strconv.FormatInt(v[5].(int64), 10),
			"function_line":  v[1].(string),
		}

		fields := map[string]interface{}{
			"section": "database",
			"query":   queryfields,
		}

		gLogger.Logger.SendDebugfLog(v[3].(string), fields)
	case "log":
		gLogger.Logger.SendInfofLog(v[3].(string), nil)
	}
}
