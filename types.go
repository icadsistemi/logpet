package logpet

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

// StandardLogger is a new type useful to add new methods for default log formats.
type StandardLogger struct {
	*logrus.Logger
	CustomFields    map[string]interface{}
	logChan         chan Log
	ddAPIKey        string
	ddEndpoint      string
	sendDebugLogs   bool
	localMode       bool
	httpClient      *http.Client
	saveOfflineLogs bool
	offlineLogsPath string
}

// Log is a type containing log message and level
type Log struct {
	Message      string
	CustomFields map[string]interface{}
	Level        logrus.Level
}

// ClientLog is a struct used for offline logs
type ClientLog struct {
	Level   string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}
