package logpet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func (l *StandardLogger) EnableOfflineLogs(enable bool) {
	l.saveOfflineLogs = enable
}

func (l *StandardLogger) saveLogToFile(toSave []byte, filename string) error {

	_, err := os.Stat(l.offlineLogsPath)
	if os.IsNotExist(err) {
		err = os.Mkdir(l.offlineLogsPath, 0644)
		if err != nil {
			return err
		}
	}

	filename = strings.ReplaceAll(filename, ":", "-")

	filename = filepath.Join(l.offlineLogsPath, filename)

	_, err = os.Stat(filename)
	if os.IsNotExist(err) {
		_, err := os.Create(filename)
		if err != nil {
			return err
		}
	}

	err = ioutil.WriteFile(filename, toSave, 0644)
	if err != nil {
		return err
	}
	return nil
}

// SendOfflineLogs send the offline logs but without extra information
// Deprecated: use SendOfflineLogsV2, new version able to filter logs and re-send as original log
func (l *StandardLogger) SendOfflineLogs() error {

	dir, err := ioutil.ReadDir(l.offlineLogsPath)
	if err != nil {
		return errors.New(fmt.Sprintf("unable to open directory %s, %v", l.offlineLogsPath, err))
	}

	for _, logfile := range dir {
		if strings.HasPrefix(logfile.Name(), "log-") {

			filename := filepath.Join(l.offlineLogsPath, logfile.Name())

			// read and send
			file, err := os.Open(filename)
			if err != nil {
				l.SendErrfLog("unable to open file %s", nil, logfile.Name())
				continue
			}

			var logs ClientLog

			err = json.NewDecoder(file).Decode(&logs)
			if err != nil {
				l.SendErrfLog("error decoding json log %s, due to %v", nil, file.Name(), err)

				// Close file and handle err
				err = file.Close()
				if err != nil {
					l.SendErrfLog("error closing json log %s, due to %v", nil, file.Name(), err)
				}
				continue
			}

			switch logs.Level {
			case "info":
				l.SendInfoLog(logs.Message, nil)
			case "warning":
				l.SendWarnLog(logs.Message, nil)
			case "error":
				l.SendErrLog(logs.Message, nil)
			case "debug":
				l.SendDebugLog(logs.Message, nil)
			case "fatal":
				l.SendErrfLog("EX FATAL | %s", nil, logs.Message)
			}

			// Close file and handle err
			err = file.Close()
			if err != nil {
				l.SendErrfLog("unable to close file %s due to %v", nil, file.Name(), err)
			}

			err = os.Remove(filename)
			if err != nil {
				l.SendErrfLog("unable to remove file %s due to %v", nil, file.Name(), err)
			}
		}
	}

	return nil
}

// SendOfflineLogsV2 iterate the offline logs and try to send them to datadog
// startingFrom param is used as a filter to send to datadog only the logs greater than that date
// the logs older than startingFrom will be deleted without sending them
func (l *StandardLogger) SendOfflineLogsV2(startingFrom time.Time) error {

	// if localMode is true, this function shouldn't be call, return an error
	if l.localMode {
		return fmt.Errorf("ReplayOfflineLog | logger localMode is set to true, cannot send offline log")
	}

	// read the files
	files, err := ioutil.ReadDir(l.offlineLogsPath)
	if err != nil {
		return fmt.Errorf("SendOfflineLogsV2 | error during readRir at path %s: %v", l.offlineLogsPath, err)
	}

	// iterate files to parse them and extract informations
	for _, file := range files {
		name := file.Name()

		// filter by .json extension and "log-" prefix
		if filepath.Ext(name) != ".json" || !strings.HasPrefix(name, LOG_FILE_PREFIX) {
			l.SendErrfLog("SendOfflineLogsV2 | skip file %s because has not .json extension or properly name", nil, name)
			continue
		}

		// extract the date part: remove prefix "log-" and suffix ".json"
		datePart := strings.TrimPrefix(name, LOG_FILE_PREFIX)
		datePart = strings.TrimSuffix(datePart, ".json")

		// parse the date
		parsedDate, err := time.Parse(LOG_FILE_DATE_LAYOUT, datePart)
		if err != nil {
			l.SendErrfLog("SendOfflineLogsV2 | skip file %s because unable to parse date from its name: %v", nil, name, err)
			continue
		}

		filePath := filepath.Join(l.offlineLogsPath, name)

		// if the log older than startingFrom, send it
		if parsedDate.After(startingFrom) {
			// read file content
			logRawContent, err := os.ReadFile(filePath)
			if err != nil {
				l.SendErrfLog("SendOfflineLogsV2 | failed to read file %s: %v", nil, filePath, err)
				continue
			}

			// try to send to datadog
			err = l.ReplayOfflineLog(string(logRawContent))
			if err != nil {
				l.SendErrfLog("SendOfflineLogsV2 | failed to resend offline log %s: %v", nil, filePath, err)
				continue
			}
		}

		// delete the log
		err = os.Remove(filePath)
		if err != nil {
			l.SendErrfLog("SendOfflineLogsV2 | error deleting file %s: %v", nil, filePath, err)
			continue
		}

	}

	return nil
}

// ReplayOfflineLog try to send an offline log to DD starting from the raw json offline log
// never ignore the debug level, because if the file is present, means that should be send to DD
func (l *StandardLogger) ReplayOfflineLog(rawJSON string) error {

	// if localMode is true, this function shouldn't be call, return an error
	if l.localMode {
		return fmt.Errorf("ReplayOfflineLog | logger localMode is set to true, cannot send offline log")
	}

	var offlineLog OfflineLog
	if err := json.Unmarshal([]byte(rawJSON), &offlineLog); err != nil {
		return fmt.Errorf("ReplayOfflineLog | failed to unmarshal raw json: %w", err)
	}

	// put on the log the original time
	entry := l.Logger.WithTime(offlineLog.ClientTime)

	// put on the log the original custom fields
	if len(offlineLog.CustomFields) > 0 {
		entry = entry.WithFields(logrus.Fields(offlineLog.CustomFields))
	}

	entry.Message = offlineLog.Message
	switch offlineLog.Level {
	case "info":
		entry.Level = logrus.InfoLevel
	case "warning":
		entry.Level = logrus.WarnLevel
	case "error":
		entry.Level = logrus.ErrorLevel
	case "debug":
		entry.Level = logrus.DebugLevel
	case "fatal":
		entry.Level = logrus.FatalLevel
	default:
		return fmt.Errorf("ReplayOfflineLog | offline log level %s not matching any logrus standard levels", offlineLog.Level)
	}

	entry.Data["ddsource"] = "logpet"

	err := l.sendLogToDD(entry, l.httpClient)
	if err != nil {
		return fmt.Errorf("ReplayOfflineLog | error sending log to datadog: %v", err)
	}

	return nil
}
