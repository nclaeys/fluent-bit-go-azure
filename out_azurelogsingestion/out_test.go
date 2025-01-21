package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestConvertToFluentbitEntry_doesNotUnwrapLogEntry(t *testing.T) {
	now := time.Now().UTC()
	log := createSimpleLog(now)
	entry := convertToFluentbitLogEntry(log, now)

	assert.Equal(t, now.Format(time.RFC3339), entry.Time)
	assert.Equal(t, "stdout", entry.Stream)
	assert.Equal(t, "{\"level\":\"debug\",\"message\":\"[azurelogsingestion] id = 0\"}", entry.Log)
}

func TestConvertToFluentbitEntry_KubernetesEntries_unwrapsThem(t *testing.T) {
	now := time.Now().UTC()
	log := createLogWithKubernetesEntries(now)
	entry := convertToFluentbitLogEntry(log, now)

	assert.Equal(t, now.Format(time.RFC3339), entry.Time)
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

func TestConvertToFluentbitEntry_handlesByteArrays(t *testing.T) {
	now := time.Now().UTC()
	log := createLogWithByteArrayValues(now)
	entry := convertToFluentbitLogEntry(log, now)

	assert.Equal(t, now.Format(time.RFC3339), entry.Time)
	assert.Equal(t, "{\"level\":\"debug\",\"message\":\"[azurelogsingestion] id = 0\"}", entry.Log)
	assert.Equal(t, "stdout", entry.Stream)
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
