package logpet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
