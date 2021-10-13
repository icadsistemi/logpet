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
	fields := map[string]interface{}{
		"section": "database",
	}

	if nil == v {
		gLogger.Logger.SendDebugLog("v database arguments are empty", fields)
		return
	}

	switch v[0] {
	case "sql":

		if len(v) != 6 {
			gLogger.Logger.SendErrfLog("wrong db log format, %v", fields, v)
			return
		}

		queryfields := map[string]string{
			"execution_time": v[2].(time.Duration).String(),
			"arguments":      fmt.Sprintf("%v", v[4]),
			"rows_affected":  strconv.FormatInt(v[5].(int64), 10),
			"function_line":  v[1].(string),
		}

		fields["query"] = queryfields

		gLogger.Logger.SendDebugfLog(v[3].(string), fields)
	case "log":
		if len(v) != 3 {
			gLogger.Logger.SendErrfLog("wrong db log format, %v", fields, v)
			return
		}

		queryfields := map[string]string{
			"function_line": v[1].(string),
		}

		fields["query"] = queryfields
		gLogger.Logger.SendWarnfLog("%v", fields, v[2])
	}
}
