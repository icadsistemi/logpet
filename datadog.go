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

func (l *StandardLogger) SetupDataDogLogger(datadogEndpoint, datadogAPIKey, offlineLogsPath string, sendDebugLogs, localmode bool) error {

	// if provided endpoint is empty we fallback to the default one
	if datadogEndpoint == "" {
		datadogEndpoint = DataDogDefaultEndpoint
	}

	if datadogAPIKey == "" && !localmode {
		return fmt.Errorf("no API Key provided")
	}

	// initialize log channel only if it doesn't exist so we don't create multiple channels
	if l.logChan == nil {
		l.initChannel()
	}

	// offline logs path
	if offlineLogsPath != "" {
		l.offlineLogsPath = offlineLogsPath
		l.EnableOfflineLogs(true)
	}

	// set debug mode with provided value
	l.SetDebugMode(sendDebugLogs)

	// enable local mode based on provided value
	l.EnableLocalMode(localmode)

	l.SetDataDogAPIKey(datadogAPIKey)

	l.SetDataDogEndpoint(datadogEndpoint)

	l.httpClient = &http.Client{}

	// starting log routine
	go l.startLogRoutineListener()

	return nil
}

func (l *StandardLogger) initChannel() {
	l.logChan = make(chan Log)
}

// EnableLocalMode assign the provided value to the client, if true it only prints log lines to the stdout
func (l *StandardLogger) EnableLocalMode(local bool) {
	l.localMode = local
}

// SetDebugMode assign the provided value to the client, if true sends and prints to stdout debug logs
func (l *StandardLogger) SetDebugMode(debug bool) {
	l.sendDebugLogs = debug
}

// SetDataDogEndpoint assign the provided datadog endpoint value to the client
func (l *StandardLogger) SetDataDogEndpoint(endpoint string) {
	l.ddEndpoint = endpoint
}

// SetDataDogAPIKey assign the provided datadog API Key value to the client
func (l *StandardLogger) SetDataDogAPIKey(APIKey string) {
	l.ddAPIKey = APIKey
}

// SetUpCustomHTTPClient assign the provided http client to the client
func (l *StandardLogger) SetUpCustomHTTPClient(httpClient *http.Client) error {
	if httpClient != nil {
		l.httpClient = httpClient
		return nil
	}

	return errors.New("nil http client provided")
}

// SendInfoLog sends a log with info level to the log channel
func (l *StandardLogger) SendInfoLog(message string, customFields map[string]interface{}) {
	go func() {
		l.logChan <- Log{
			Message:      message,
			CustomFields: customFields,
			Level:        logrus.InfoLevel,
		}
	}()
}

// SendInfofLog sends a formatted log with info level to the log channel
func (l *StandardLogger) SendInfofLog(message string, customFields map[string]interface{}, args ...interface{}) {
	l.SendInfoLog(fmt.Sprintf(message, args...), customFields)
}

// SendWarnLog sends a log with warning level to the log channel
func (l *StandardLogger) SendWarnLog(message string, customFields map[string]interface{}) {
	go func() {
		l.logChan <- Log{
			Message:      message,
			CustomFields: customFields,
			Level:        logrus.WarnLevel,
		}
	}()
}

// SendWarnfLog sends a formatted log with warn level to the log channel
func (l *StandardLogger) SendWarnfLog(message string, customFields map[string]interface{}, args ...interface{}) {
	l.SendWarnLog(fmt.Sprintf(message, args...), customFields)
}

// SendErrLog sends a log with error level to the log channel
func (l *StandardLogger) SendErrLog(message string, customFields map[string]interface{}) {
	go func() {
		l.logChan <- Log{
			Message:      message,
			CustomFields: customFields,
			Level:        logrus.ErrorLevel,
		}
	}()
}

// SendErrfLog sends a formatted log with error level to the log channel
func (l *StandardLogger) SendErrfLog(message string, customFields map[string]interface{}, args ...interface{}) {
	l.SendErrLog(fmt.Sprintf(message, args...), customFields)
}

// SendDebugLog sends a log with debug level to the log channel
func (l *StandardLogger) SendDebugLog(message string, customFields map[string]interface{}) {
	go func() {
		l.logChan <- Log{
			Message:      message,
			CustomFields: customFields,
			Level:        logrus.DebugLevel,
		}
	}()
}

// SendDebugfLog sends a formatted log with debug level to the log channel
func (l *StandardLogger) SendDebugfLog(message string, customFields map[string]interface{}, args ...interface{}) {
	l.SendDebugLog(fmt.Sprintf(message, args...), customFields)
}

// SendFatalLog sends a log with fatal level to the log channel
func (l *StandardLogger) SendFatalLog(message string, customFields map[string]interface{}) {
	func() {
		l.logChan <- Log{
			Message:      message,
			CustomFields: customFields,
			Level:        logrus.FatalLevel,
		}
	}()
}

// SendFatalfLog sends a formatted log with fatal level to the log channel
func (l *StandardLogger) SendFatalfLog(message string, customFields map[string]interface{}, args ...interface{}) {
	l.SendFatalLog(fmt.Sprintf(message, args...), customFields)
}

// startLogRoutineListener handles the incoming logs
func (l *StandardLogger) startLogRoutineListener() {
	var logWriter io.Writer
	l.SetOutput(logWriter)

	for logElem := range l.logChan {

		// ignore debug log if sendDebugLog is set to false
		if !l.sendDebugLogs && logElem.Level == logrus.DebugLevel {
			continue
		}

		l.processLogElem(logElem)
	}
}

// processLogElem builds, dispatches and (if needed) terminates the process
// for a single log entry. The defer guarantees that a Fatal log always
// triggers os.Exit(1), even if the dispatch to Datadog fails or an early
// return/continue-equivalent happens inside this function.
func (l *StandardLogger) processLogElem(logElem Log) {

	// in case of FatalLevel log, exit after function ending
	defer func() {
		if logElem.Level == logrus.FatalLevel {
			os.Exit(1)
		}
	}()

	newLog := l.AddCustomFields()
	newLog.Message = logElem.Message
	newLog.Level = logElem.Level
	newLog.Time = time.Now()

	newLog.Data["ddsource"] = "logpet"

	for key, value := range logElem.CustomFields {
		newLog.Data[key] = value
	}

	logBytes, err := newLog.Bytes()
	if err != nil {
		l.SendWarnLog(fmt.Sprintf("error converting log to bytes %v", err), nil)
		return
	}

	// If localMode is true print the log with Println
	if l.localMode {
		fmt.Println(string(logBytes))
		return
	}

	err = l.sendLogToDD(newLog, l.httpClient)
	if err == nil {
		return
	}

	log.Printf("unable to send log to DataDog, %v", err)

	if !l.saveOfflineLogs {
		return
	}

	newLog.Message = fmt.Sprintf("OFFLINE LOG at %v | %s", time.Now().String(), newLog.Message)

	logBytes, offsaveErr := newLog.Bytes()
	if offsaveErr != nil {
		l.SendWarnLog(fmt.Sprintf("error converting log to bytes %v", offsaveErr), nil)
		return
	}

	if offsaveErr = l.saveLogToFile(logBytes, fmt.Sprintf("log-%s.json", time.Now().Format(time.RFC3339Nano))); offsaveErr != nil {
		fmt.Println(offsaveErr)
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
		return fmt.Errorf("error when sending logs to DD | Status: %s %v", resp.Status, err)
	}

	return nil
}
