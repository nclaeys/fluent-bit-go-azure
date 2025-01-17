package main

import (
	"C"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
)

var azureLogOperators []*AzureOperator

type AzureConfig struct {
	WorkspaceId string
	SharedKey   string
	LogType     string
	EndpointURI string
	LogLevel    string
}

type AzureOperator struct {
	config AzureConfig
	client *http.Client
}

//export FLBPluginRegister
func FLBPluginRegister(def unsafe.Pointer) int {
	log.Debug().Msg("[azureconveyor] Register called")
	return output.FLBPluginRegister(def, "azureconveyor", "Registering azureconveyor output.")
}

//export FLBPluginInit
func FLBPluginInit(plugin unsafe.Pointer) int {
	operatorID := len(azureLogOperators)
	log.Debug().Msgf("[azureconveyor] id = %d", operatorID)
	output.FLBPluginSetContext(plugin, operatorID)
	operator, err := createAzureOperator(plugin)
	if err != nil {
		log.Err(err).Msg("failed creating azure operator")
		output.FLBPluginUnregister(plugin)
		os.Exit(1)
		return output.FLB_ERROR
	}
	azureLogOperators = append(azureLogOperators, operator)

	return output.FLB_OK
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	log.Debug().Msg("[azureconveyor] Flush called for unknown instance")
	return output.FLB_OK
}

//export FLBPluginExit
func FLBPluginExit() int {
	log.Debug().Msg("[azureconveyor] Exit called for unknown instance")
	return output.FLB_OK
}

//export FLBPluginExitCtx
func FLBPluginExitCtx(ctx unsafe.Pointer) int {
	id := output.FLBPluginGetContext(ctx).(string)
	log.Debug().Msgf("[azureconveyor] Exit called for id: %s", id)
	return output.FLB_OK
}

//export FLBPluginUnregister
func FLBPluginUnregister(def unsafe.Pointer) {
	log.Debug().Msg("[azureconveyor] Unregister called")
	output.FLBPluginUnregister(def)
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	id := output.FLBPluginGetContext(ctx).(int)
	log.Debug().Msgf("[azureconveyor] Flush called for id: %d", id)
	operator := azureLogOperators[id]
	decoder := output.NewDecoder(data, int(length))
	marshalledValue, err := convertBinaryDataToJson(decoder)
	if err != nil {
		return output.FLB_ERROR
	}
	err = operator.SendLogs(marshalledValue)
	if err != nil {
		return output.FLB_RETRY
	}
	return output.FLB_OK
}

func (a *AzureOperator) BuildHeaders(contentAsString string) (map[string]string, error) {
	date := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")

	azureApiSigningString := "POST\n" + strconv.Itoa(utf8.RuneCountInString(contentAsString)) + "\napplication/json\nx-ms-date:" + date + "\n/api/logs"
	keyBytes, err := base64.StdEncoding.DecodeString(a.config.SharedKey)
	if err != nil {
		return nil, err
	}

	h := hmac.New(sha256.New, keyBytes)
	h.Write([]byte(azureApiSigningString))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	headers := map[string]string{
		"Content-Type":  "application/json",
		"x-ms-date":     date,
		"Authorization": fmt.Sprintf("SharedKey %s:%s", a.config.WorkspaceId, signature),
		"Log-Type":      a.config.LogType,
		"User-Agent":    "Fluent-Bit",
	}

	return headers, nil
}

func createAzureOperator(plugin unsafe.Pointer) (*AzureOperator, error) {
	workspaceId := output.FLBPluginConfigKey(plugin, "workspace_id")
	sharedKey := output.FLBPluginConfigKey(plugin, "shared_key")
	logType := output.FLBPluginConfigKey(plugin, "log_type")
	logLevel := output.FLBPluginConfigKey(plugin, "log_level")
	config := AzureConfig{
		WorkspaceId: workspaceId,
		SharedKey:   sharedKey,
		LogType:     logType,
		LogLevel:    logLevel,
		EndpointURI: constructEndpointUri(workspaceId),
	}
	if logLevel != "" {
		level, err := zerolog.ParseLevel(logLevel)
		if err != nil {
			log.Err(errors.Wrap(err, "failed to parse log level"))
			return nil, err
		}
		zerolog.SetGlobalLevel(level)
	} else {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}

	log.Info().Msgf("[azureconveyor] Config: %v", config)
	return &AzureOperator{
		config: config,
		client: &http.Client{},
	}, nil
}

func constructEndpointUri(customerId string) string {
	return fmt.Sprintf("https://%s.ods.opinsights.azure.com/%s?api-version=2016-04-01", customerId, "api/logs")
}

func convertBinaryDataToJson(dec *output.FLBDecoder) (string, error) {
	var jsonEntries []map[string]interface{}
	count := 0
	for {
		ret, ts, record := output.GetRecord(dec)
		if ret != 0 {
			break
		}
		var timestamp time.Time
		switch t := ts.(type) {
		case output.FLBTime:
			timestamp = ts.(output.FLBTime).Time
		case uint64:
			timestamp = time.Unix(int64(t), 0)
		default:
			log.Debug().Msg("time provided invalid, defaulting to now.")
			timestamp = time.Now()
		}
		timestampString := timestamp.UTC().Format(time.RFC3339)
		record["time"] = timestampString
		jsonStruct := encodeJSON(record)
		jsonEntries = append(jsonEntries, jsonStruct)
		count++
	}
	marshalledValue, err := json.Marshal(jsonEntries)
	if err != nil {
		log.Err(err).Msg("[azureconveyor] Failed ot marshal json")
		return "", err
	}
	log.Debug().Msgf("[azureconveyor] converted %d logs", count)
	return string(marshalledValue), nil
}

func (a *AzureOperator) SendLogs(value string) error {
	headers, err := a.BuildHeaders(value)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", a.config.EndpointURI, bytes.NewReader([]byte(value)))
	if err != nil {
		log.Err(err).Msg("[azureconveyor] Failed to contruct request")
		return err
	}
	for k, v := range headers {
		request.Header.Set(k, v)
	}
	resp, err := a.client.Do(request)
	if err != nil {
		log.Err(err).Msg("[azureconveyor] Request to azure endpoint failed")
		return err
	}
	if resp.StatusCode > 299 {
		log.Warn().Msgf("[azureconveyor] Request to endpoint failed with status code: %d", resp.StatusCode)
		if resp.ContentLength > 0 {
			_, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Info().Msgf("[azureconveyor] Failed to read response body: %s", err.Error())
			}
		}
		return errors.New("request failed")
	} else if resp.StatusCode == 200 {
		log.Debug().Msg("[azureconveyor] Successful request send to azure")
	}
	return nil
}

func encodeJSON(record map[interface{}]interface{}) map[string]interface{} {
	m := make(map[string]interface{})

	for k, v := range record {
		switch t := v.(type) {
		case []byte:
			// prevent encoding to base64
			m[k.(string)] = string(t)
		case map[interface{}]interface{}:
			if nextValue, ok := record[k].(map[interface{}]interface{}); ok {
				m[k.(string)] = encodeJSON(nextValue)
			}
		default:
			m[k.(string)] = v
		}
	}

	return m
}

func main() {
}
