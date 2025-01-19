package main

import (
	"C"
	"context"
	"encoding/json"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
)

var azureLogOperators []*AzureOperator

type FluentbitLogEntry struct {
	TimeGenerated            string `json:"TimeGenerated"`
	Time                     string `json:"time"`
	KubernetesPodName        string `json:"kubernetes_pod_name"`
	KubernetesPodId          string `json:"kubernetes_pod_id"`
	KubernetesNamespaceName  string `json:"kubernetes_namespace_name"`
	KubernetesHost           string `json:"kubernetes_host"`
	KubernetesDockerId       string `json:"kubernetes_docker_id"`
	KubernetesContainerName  string `json:"kubernetes_container_name"`
	KubernetesContainerImage string `json:"kubernetes_container_image"`
	KubernetesContainerHash  string `json:"kubernetes_container_hash"`
	Log                      string `json:"log"`
	Stream                   string `json:"stream"`
}

type AzureConfig struct {
	DcrImmutableId string
	Endpoint       string
	StreamName     string
	EndpointURI    string
	LogLevel       string
}

type AzureOperator struct {
	config     AzureConfig
	logsClient *azlogs.Client
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

	jsonResult, err := convertToJson(decoder)
	if err != nil {
		return output.FLB_ERROR
	}
	err = operator.SendLogs(jsonResult)
	if err != nil {
		return output.FLB_RETRY
	}

	return output.FLB_OK
}

func createAzureOperator(plugin unsafe.Pointer) (*AzureOperator, error) {
	dcrImmutableId := output.FLBPluginConfigKey(plugin, "dcr_immutable_id")
	endpoint := output.FLBPluginConfigKey(plugin, "endpoint")
	streamName := output.FLBPluginConfigKey(plugin, "stream_name")
	logLevel := output.FLBPluginConfigKey(plugin, "log_level")
	config := AzureConfig{
		DcrImmutableId: dcrImmutableId,
		Endpoint:       endpoint,
		StreamName:     streamName,
		LogLevel:       logLevel,
	}
	err := setLogLevel(logLevel)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("[azureconveyor] Config: %v", config)
	return &AzureOperator{
		config:     config,
		logsClient: constructClient(config),
	}, nil
}

func setLogLevel(logLevel string) error {
	if logLevel != "" {
		level, err := zerolog.ParseLevel(logLevel)
		if err != nil {
			log.Err(errors.Wrap(err, "failed to parse log level"))
			return err
		}
		zerolog.SetGlobalLevel(level)
		return nil
	}
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	return nil
}

func constructClient(config AzureConfig) *azlogs.Client {
	var cred, err = azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		panic(err)
	}

	client, err := azlogs.NewClient(config.Endpoint, cred, nil)
	if err != nil {
		panic(err)
	}
	return client
}

func convertToJson(dec *output.FLBDecoder) (string, error) {
	var jsonEntries []FluentbitLogEntry
	count := 0
	for {
		ret, ts, record := output.GetRecord(dec)
		if ret != 0 {
			break
		}
		fluentbitEntry := convertToFluentbitLogEntry(record, getTimestampOrNow(ts))
		jsonEntries = append(jsonEntries, fluentbitEntry)
		count++
	}
	marshalledValue, err := json.Marshal(jsonEntries)
	if err != nil {
		log.Err(err).Msg("[azureconveyor] Failed ot marshal fluentbit entries to json")
		return "", err
	}
	log.Debug().Msgf("[azureconveyor] converted %d logs", count)
	return string(marshalledValue), nil
}

func getTimestampOrNow(ts interface{}) time.Time {
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
	return timestamp
}

func convertToFluentbitLogEntry(record map[interface{}]interface{}, timestamp time.Time) FluentbitLogEntry {
	fluentBitLog := FluentbitLogEntry{
		Time: timestamp.UTC().Format(time.RFC3339),
	}
	for k, v := range record {
		key := k.(string)
		value := v.(string)
		switch key {
		case "kubernetes_pod_name":
			fluentBitLog.KubernetesPodName = value
		case "kubernetes_pod_id":
			fluentBitLog.KubernetesPodId = value
		case "kubernetes_namespace_name":
			fluentBitLog.KubernetesNamespaceName = value
		case "kubernetes_host":
			fluentBitLog.KubernetesHost = value
		case "kubernetes_docker_id":
			fluentBitLog.KubernetesDockerId = value
		case "kubernetes_container_name":
			fluentBitLog.KubernetesContainerName = value
		case "kubernetes_container_image":
			fluentBitLog.KubernetesContainerImage = value
		case "kubernetes_container_hash":
			fluentBitLog.KubernetesContainerHash = value
		case "log":
			fluentBitLog.Log = value
		case "stream":
			fluentBitLog.Stream = value
		default:
			log.Debug().Msgf("[azureconveyor] Unknown record key: %s", key)
		}
	}
	return fluentBitLog
}

func (a *AzureOperator) SendLogs(value string) error {
	_, err := a.logsClient.Upload(context.Background(),
		a.config.DcrImmutableId,
		a.config.StreamName,
		[]byte(value),
		nil)
	return err
}

func main() {
}
