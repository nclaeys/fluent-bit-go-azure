package main

import (
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
	mock_logs "github.com/fluent/fluent-bit-go/out_azurelogsingestion/mocks/azlogs/mock_logsclient"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
)

func TestProcessEntries_nil_noError(t *testing.T) {
	err := processEntries(nil, nil)

	assert.NoError(t, err)
}

func TestConvertToFluentbitLogEntry_doesNotUnwrapLogEntry(t *testing.T) {
	now := time.Now().UTC()
	log := createSimpleLog(now)
	entry := convertToFluentbitLogEntry(log, now)

	assert.Equal(t, now.Format(time.RFC3339Nano), entry.TimeGenerated)
	assert.Equal(t, "stdout", entry.Stream)
	assert.Equal(t, "{\"level\":\"debug\",\"message\":\"[azurelogsingestion] id = 0\"}", entry.Log)
}

func TestConvertToFluentbitLogEntry_KubernetesEntries_unwrapsThem(t *testing.T) {
	now := time.Now().UTC()
	log := createLogWithKubernetesEntries(now)
	entry := convertToFluentbitLogEntry(log, now)

	assert.Equal(t, now.Format(time.RFC3339Nano), entry.TimeGenerated)
	assert.Equal(t, "{\"level\":\"debug\",\"message\":\"[azurelogsingestion] id = 0\"}", entry.Log)
	assert.Equal(t, "container_name", entry.KubernetesContainerName)
	assert.Equal(t, "pod_name", entry.KubernetesPodName)
	assert.Equal(t, "container_image", entry.KubernetesContainerImage)
	assert.Equal(t, "container_hash", entry.KubernetesContainerHash)
	assert.Equal(t, "host", entry.KubernetesHost)
	assert.Equal(t, "docker_id", entry.KubernetesDockerId)
	assert.Equal(t, "pod_id", entry.KubernetesPodId)
	assert.Equal(t, "namespace_name", entry.KubernetesNamespaceName)
}

func TestConvertToFluentbitLogEntry_handlesByteArrays(t *testing.T) {
	now := time.Now().UTC()
	log := createLogWithByteArrayValues(now)
	entry := convertToFluentbitLogEntry(log, now)

	assert.Equal(t, now.Format(time.RFC3339Nano), entry.TimeGenerated)
	assert.Equal(t, "{\"level\":\"debug\",\"message\":\"[azurelogsingestion] id = 0\"}", entry.Log)
	assert.Equal(t, "stdout", entry.Stream)
}

func TestConvertFluentbitEntriesToJson_emptyList_returnsEmptyList(t *testing.T) {
	entry, err := convertFluentbitEntriesToJson([]FluentbitLogEntry{})

	assert.NoError(t, err)
	assert.Len(t, entry, 0)
}

func TestConvertFluentbitEntriesToJson_returnsJsonResult(t *testing.T) {
	log := generateDummyFluentbitLogEntry()
	entry, err := convertFluentbitEntriesToJson([]FluentbitLogEntry{log, log})

	assert.NoError(t, err)
	assert.Len(t, entry, 1)
}

func TestConvertFluentbitEntriesToJson_normalEntriesLargerThan1Megabyte_splitsUpResult(t *testing.T) {
	log := generateDummyFluentbitLogEntry()
	var entriesLargerOneMb []FluentbitLogEntry
	for range 1400 {
		entriesLargerOneMb = append(entriesLargerOneMb, log)
	}
	entry, err := convertFluentbitEntriesToJson(entriesLargerOneMb)

	assert.NoError(t, err)
	assert.Len(t, entry, 2)
}

func TestConvertFluentbitEntriesToJson_bigEntriesLargerThan1Megabyte_splitsUpResult(t *testing.T) {
	longLog := "[2025-05-12 12:12:27,166] {kubernetes_executor.py:380} DEBUG - self.running: {TaskInstanceKey(dag_id='azurepython-secrets-fail', task_id='secrets-keyvault-parameter-does-not-exist', run_id='scheduled__2025-05-11T00:00:00+00:00', try_number=1, map_index=-1), TaskInstanceKey(dag_id='azurepython-secrets-fail', task_id='secrets-client-id-not-exists', run_id='scheduled__2025-05-11T00:00:00+00:00', try_number=1, map_index=-1), TaskInstanceKey(dag_id='azurepython-secrets-fail', task_id='secrets-client-id-no-identity-credential', run_id='scheduled__2025-05-11T00:00:00+00:00', try_number=1, map_index=-1), TaskInstanceKey(dag_id='azurepython-secrets-fail', task_id='secrets-no-client-id', run_id='scheduled__2025-05-11T00:00:00+00:00', try_number=1, map_index=-1), TaskInstanceKey(dag_id='azurepython-secrets-fail', task_id='secrets-client-id-no-keyvault-access', run_id='scheduled__2025-05-11T00:00:00+00:00', try_number=1, map_index=-1)}"
	longLogEntry := generateDummyFluentbitLogEntryWithLog(longLog)
	var entriesLargerOneMb []FluentbitLogEntry
	for range 750 {
		entriesLargerOneMb = append(entriesLargerOneMb, longLogEntry)
	}
	entry, err := convertFluentbitEntriesToJson(entriesLargerOneMb)

	assert.NoError(t, err)
	assert.Len(t, entry, 2)
}

func generateDummyFluentbitLogEntry() FluentbitLogEntry {
	log := "exec /usr/bin/tini -s -- /opt/spark/bin/spark-submit --master k8s://https://datafy-dp-deva-nc-dev-q0xnpayz.hcp.westeurope.azmk8s.io --name 4420609e-2fde-48f4-a686-499c0262f5a2 --deploy-mode cluster"
	return generateDummyFluentbitLogEntryWithLog(log)
}

func generateDummyFluentbitLogEntryWithLog(log string) FluentbitLogEntry {
	return FluentbitLogEntry{
		TimeGenerated:            time.Now().UTC().Format(time.RFC3339Nano),
		KubernetesPodName:        "datafy-pyspark-sample-b7b8ff96c4335653-exec-1",
		KubernetesPodId:          "cf8291a4-5d2f-4f54-9466-da36a3899dd9",
		KubernetesNamespaceName:  "namespace",
		KubernetesHost:           "aks-sd8sv51313-14978311-vmss000004",
		KubernetesDockerId:       "aks-sd8sv51313-14978311-vmss000004",
		KubernetesContainerName:  "my-base-container",
		KubernetesContainerImage: "my-container-image",
		KubernetesContainerHash:  "f9519bbaf68aab771291618e8245b43dc0990a1df3e69eafd04c760c54c27aba",
		Log:                      log,
		Stream:                   "info",
	}
}

func createLogWithByteArrayValues(now time.Time) map[interface{}]interface{} {
	return map[interface{}]interface{}{
		"time":   now,
		"stream": []byte("stdout"),
		"_p":     "F",
		"log":    []byte("{\"level\":\"debug\",\"message\":\"[azurelogsingestion] id = 0\"}"),
	}
}

func createLogWithKubernetesEntries(now time.Time) map[interface{}]interface{} {
	return map[interface{}]interface{}{
		"time":   now,
		"stream": "stdout",
		"_p":     "F",
		"log":    "{\"level\":\"debug\",\"message\":\"[azurelogsingestion] id = 0\"}",
		"kubernetes": map[interface{}]interface{}{
			"pod_name":        "pod_name",
			"pod_id":          "pod_id",
			"namespace_name":  "namespace_name",
			"host":            "host",
			"docker_id":       "docker_id",
			"container_name":  "container_name",
			"container_image": "container_image",
			"container_hash":  "container_hash",
		},
	}
}

func createSimpleLog(now time.Time) map[interface{}]interface{} {
	return map[interface{}]interface{}{
		"time":   now,
		"stream": "stdout",
		"_p":     "F",
		"log":    "{\"level\":\"debug\",\"message\":\"[azurelogsingestion] id = 0\"}",
	}
}

func TestSendLogs_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_logs.NewMockAzureLogsClient(ctrl)
	operator := &AzureOperator{
		config: AzureConfig{
			DcrImmutableId: "test-id",
			StreamName:     "test-stream",
		},
		logsClient: mockClient,
	}

	mockClient.EXPECT().Upload(gomock.Any(), "test-id", "test-stream", gomock.Any(), gomock.Any()).Return(azlogs.UploadResponse{}, nil)

	err := operator.SendLogs(`{"log": "test message"}`)
	assert.NoError(t, err)
}
