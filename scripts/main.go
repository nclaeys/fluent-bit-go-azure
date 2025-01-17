package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
)

// logs ingestion URI
const endpoint = "https://k8slogsncdev-pfzu.westeurope-1.ingest.monitor.azure.com"

// data collection rule (DCR) immutable ID
const ruleID = "dcr-89086b086dce443da28f0c304e7d49fb"

// stream name in the DCR that represents the destination table
const streamName = "Custom-fluentbit-json-stream"

// Tenant ID for the Azure AD application
const tenantID = ""

// Client secret of the Azure AD application with access to the DCR
const clientSecret = ""

// Client ID of the Azure AD application with access to the DCR
const clientID = ""

type FluentBitLog struct {
	TimeGenerated            time.Time `json:"TimeGenerated"`
	Time                     time.Time `json:"time"`
	KubernetesPodName        string    `json:"kubernetes_pod_name"`
	KubernetesPodId          string    `json:"kubernetes_pod_id"`
	KubernetesNamespace      string    `json:"kubernetes_namespace"`
	KubernetesHost           string    `json:"kubernetes_host"`
	KubernetesDockerId       string    `json:"kubernetes_docker_id"`
	KubernetesContainerName  string    `json:"kubernetes_container_name"`
	KubernetesContainerImage string    `json:"kubernetes_container_image"`
	KubernetesContainerHash  string    `json:"kubernetes_container_hash"`
	Log                      string    `json:"log"`
	Stream                   string    `json:"stream"`
}

func main() {
	var cred, err = azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		panic(err)
	}

	client, err := azlogs.NewClient(endpoint, cred, nil)
	if err != nil {
		panic(err)
	}

	var data []FluentBitLog

	for i := 0; i < 10; i++ {
		data = append(data, FluentBitLog{
			Time:                     time.Now().UTC(),
			TimeGenerated:            time.Now().UTC(),
			KubernetesNamespace:      "default",
			KubernetesContainerHash:  "someHash",
			KubernetesPodId:          "podId",
			Log:                      "someLog",
			KubernetesContainerImage: "containerImage",
			KubernetesHost:           "host",
			KubernetesContainerName:  "containerName",
			KubernetesPodName:        "podName",
			KubernetesDockerId:       "dockerId",
			Stream:                   "stream",
		})
	}

	logs, err := json.Marshal(data)

	if err != nil {
		panic(err)
	}

	_, err = client.Upload(context.TODO(), ruleID, streamName, logs, nil)

	if err != nil {
		panic(err)
	}
	println("Successfully uploaded logs")
}
