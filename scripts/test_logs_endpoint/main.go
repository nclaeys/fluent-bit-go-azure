package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
)

// logs ingestion URI
var endpoint = ""

// data collection rule (DCR) immutable ID
var ruleID = ""

// stream name in the DCR that represents the destination table
var streamName = ""

// Tenant ID for the Azure AD application
var tenantID = ""

// Client secret of the Azure AD application with access to the DCR
var clientSecret = ""

// Client ID of the Azure AD application with access to the DCR
var clientID = ""

type FluentBitLog struct {
	TimeGenerated            time.Time `json:"TimeGenerated"`
	Time                     time.Time `json:"time"`
	KubernetesPodName        string    `json:"kubernetes_pod_name"`
	KubernetesPodId          string    `json:"kubernetes_pod_id"`
	KubernetesNamespaceName  string    `json:"kubernetes_namespace_name"`
	KubernetesHost           string    `json:"kubernetes_host"`
	KubernetesDockerId       string    `json:"kubernetes_docker_id"`
	KubernetesContainerName  string    `json:"kubernetes_container_name"`
	KubernetesContainerImage string    `json:"kubernetes_container_image"`
	KubernetesContainerHash  string    `json:"kubernetes_container_hash"`
	Log                      string    `json:"log"`
	Stream                   string    `json:"stream"`
}

func main() {
	readDotEnvFile()
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
			KubernetesNamespaceName:  "default",
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

func readDotEnvFile() {
	_, err := os.Stat(".env")
	if os.IsNotExist(err) {
		panic(errors.New("no .env file found"))
	}
	envFile, err := os.ReadFile(".env")
	if err != nil {
		panic(errors.Wrap(err, "failed to read .env file"))
	}
	for _, line := range bytes.Split(envFile, []byte("\n")) {
		parts := bytes.Split(line, []byte("="))
		if len(parts) == 2 {
			key := string(parts[0])
			value := string(parts[1])
			setProperty(key, value)
		} else {
			println("Skipping line in .env file")
		}
	}
}

func setProperty(key string, value string) {
	switch key {
	case "client_secret":
		clientSecret = value
	case "client_id":
		clientID = value
	case "tenant_id":
		tenantID = value
	case "endpoint":
		endpoint = value
	case "dcr_immutable_id":
		ruleID = value
	case "stream_name":
		streamName = value
	}
}
