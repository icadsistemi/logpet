package logpet

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const LOG_FILE_PREFIX = "log-"
const LOG_FILE_DATE_LAYOUT = "2006-01-02T15-04-05.9999Z07-00"

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

// OfflineLog is a struct used for replay an offline log, also with original timestamp and custom fields
type OfflineLog struct {
	ClientTime   time.Time              `json:"client_time"`
	Message      string                 `json:"message"`
	Level        string                 `json:"status"`
	CustomFields map[string]interface{} `json:"-"`
}

// UnmarshalJSON is a method able to extract from a raw json the offline log structure
func (o *OfflineLog) UnmarshalJSON(data []byte) error {
	// deserialize all in a generic structure
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// extract known fields
	if v, ok := raw["message"].(string); ok {
		o.Message = v
	}
	if v, ok := raw["status"].(string); ok {
		o.Level = v
	}
	if v, ok := raw["client_time"].(string); ok {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return fmt.Errorf("failed to parse client_time: %w", err)
		}
		o.ClientTime = parsed
	}

	// delete the known fields and put the others on custom fields
	delete(raw, "message")
	delete(raw, "status")
	delete(raw, "client_time")
	o.CustomFields = raw

	return nil
}
