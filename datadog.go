package logpet

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

func (l *StandardLogger) SetupDataDogLogger(datadogEndpoint, datadogAPIKey string, sendDebugLogs, localmode bool) error {

	// if provided endpoint is empty we fallback to the default one
	if datadogEndpoint == "" {
		datadogEndpoint = DataDogDefaultEndpoint
	}

	if datadogAPIKey == "" && !localmode {
		return fmt.Errorf("no API Key provided")
	}

	// initialize log channel
	l.initChannel()

	// set debug mode with provided value
	l.SetDebugMode(sendDebugLogs)

	// enable local mode based on provided value
	l.EnableLocalMode(localmode)

	l.ddAPIKey = datadogAPIKey
	l.ddEndpoint = datadogEndpoint

	// starting log routine
	go l.startLogRoutineListener()

	return nil
}

func (l *StandardLogger) initChannel() {
	l.LogChan = make(chan Log)
}

// EnableLocalMode assign the provided value to the client, if true it only prints log lines to the stdout
func (l *StandardLogger) EnableLocalMode(local bool) {
	l.localMode = local
}

// SetDebugMode assign the provided value to the client, if true sends and prints to stdout debug logs
func (l *StandardLogger) SetDebugMode(debug bool) {
	l.sendDebugLogs = debug
}

// SendInfoLog sends a log with info level to the log channel
func (l *StandardLogger) SendInfoLog(message, customhostname string) {
	l.LogChan <- Log{
		Message:        message,
		CustomHostname: customhostname,
		Level:          logrus.InfoLevel,
	}
}

// SendWarnLog sends a log with warning level to the log channel
func (l *StandardLogger) SendWarnLog(message, customhostname string) {
	l.LogChan <- Log{
		Message:        message,
		CustomHostname: customhostname,
		Level:          logrus.WarnLevel,
	}
}

// SendErrLog sends a log with error level to the log channel
func (l *StandardLogger) SendErrLog(message, customhostname string) {
	l.LogChan <- Log{
		Message:        message,
		CustomHostname: customhostname,
		Level:          logrus.ErrorLevel,
	}
}

// SendDebugLog sends a log with debug level to the log channel
func (l *StandardLogger) SendDebugLog(message, customhostname string) {
	l.LogChan <- Log{
		Message:        message,
		CustomHostname: customhostname,
		Level:          logrus.DebugLevel,
	}
}

// SendFatalLog sends a log with fatal level to the log channel
func (l *StandardLogger) SendFatalLog(message, customhostname string) {
	l.LogChan <- Log{
		Message:        message,
		CustomHostname: customhostname,
		Level:          logrus.FatalLevel,
	}
}

// startLogRoutineListener handles the incoming logs
func (l *StandardLogger) startLogRoutineListener() {
	var logWriter io.Writer
	var httpClient http.Client
	l.SetOutput(logWriter)

	for logElem := range l.LogChan {

		// ignore debug log if sendDebugLog is set to false
		if !l.sendDebugLogs && logElem.Level == logrus.DebugLevel {
			continue
		}

		newLog := l.AddCustomFields()
		newLog.Message = logElem.Message
		newLog.Level = logElem.Level
		newLog.Time = time.Now()

		if logElem.CustomHostname != "" {
			newLog.Data["host"] = logElem.CustomHostname
		}

		// If sendDebugLogs is true print the log with Println
		if l.sendDebugLogs || l.localMode {
			logBytes, err := newLog.Bytes()
			if err != nil {
				l.SendWarnLog(fmt.Sprintf("error converting log to bytes %v", err), "")
				continue
			}

			fmt.Println(string(logBytes))
		}

		// Performing http request to datadog
		if !l.localMode {
			err := l.sendLogToDD(newLog, &httpClient)
			if err != nil {
				log.Printf("unable to send log to DataDog, %v", err)
				continue
			}
		}

		// If it's a fatal log exit
		if logElem.Level == logrus.FatalLevel {
			os.Exit(1)
		}
	}
}

func (l *StandardLogger) sendLogToDD(log *logrus.Entry, httpClient *http.Client) error {

	// obtaining byte slice from log
	logBytes, err := log.Bytes()
	if err != nil {
		return err
	}

	// creating the reader from slice
	body := bytes.NewReader(logBytes)

	// parsing datadog endpoint URL
	urlPrsd, err := url.Parse(l.ddEndpoint)
	if err != nil {
		return err
	}

	// creating new request
	req, err := http.NewRequest(http.MethodPost, urlPrsd.String(), body)
	if err != nil {
		return err
	}

	// adding apikey and content type header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", l.ddAPIKey)

	// doing the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// if not ok return an error
	if resp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("error when sending logs to DD | Status: %s %v", resp.Status, err))
	}

	return nil
}
