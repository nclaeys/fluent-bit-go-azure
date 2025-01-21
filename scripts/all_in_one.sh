#!/bin/bash

RESOURCE_GROUP=fluentbit-logs-rg
WORKSPACE_NAME=fluentbit-logs-k8s
TABLE_NAME=fluentbit_logs_k8s
DATA_COLLECTION_ENDPOINT_NAME=fluentbit_logs_endpoint
LOCATION=westeurope
IDENITY_NAME=fluentbit-k8s

cd "$(dirname "$0")"
echo "executing script from: $(pwd)"

echo "Creating resource group $RESOURCE_GROUP..."
az group create --name $RESOURCE_GROUP --location $LOCATION

echo "Creating Log Analytics workspace with name $WORKSPACE_NAME..."
#TODO check identity type
workspace_response=$(az monitor log-analytics workspace create --resource-group $RESOURCE_GROUP --name $WORKSPACE_NAME --location $LOCATION --identity-type UserAssigned --sku Free)

echo "Creating log analytics table $TABLE_NAME..."
az monitor log-analytics workspace table create --workspace-name $WORKSPACE_NAME --resource-group $RESOURCE_GROUP --name "${TABLE_NAME}_CL" \
--columns TimeGenerated=datetime kubernetes_pod_name=string kubernetes_pod_id=string kubernetes_namespace_name=string kubernetes_host=string \
kubernetes_docker_id=string kubernetes_container_name=string kubernetes_container_image=string kubernetes_container_hash=string log=string stream=string \
--plan Basic

echo "Creating data collection endpoint with name $DATA_COLLECTION_ENDPOINT_NAME..."
data_collection_endpoint_response=$(az monitor data-collection endpoint create --name $DATA_COLLECTION_ENDPOINT_NAME --resource-group $RESOURCE_GROUP --public-network-access Enabled --location $LOCATION)

workspace_id=$(echo $workspace_response | jq -r '.id')
echo "Log analytics workspaceId: $workspace_id"

data_collection_endpoint=$(echo $data_collection_endpoint_response | jq -r '.logsIngestion.endpoint')
data_collection_endpoint_id=$(echo $data_collection_endpoint_response | jq -r '.id')
echo "Data collection endpoint uri: $data_collection_endpoint"

./create-dcr/generate-dcr.sh $data_collection_endpoint_id $workspace_id $TABLE_NAME
echo "Generated data collection rule template: \n $(cat ./create-dcr/fluentbit-logs-dcr-output.json)"

echo "Creating data collection rule..."
dcr_response=$(az monitor data-collection rule create \
--location $LOCATION \
--resource-group $RESOURCE_GROUP \
--endpoint-id $data_collection_endpoint \
--rule-file ./create-dcr/fluentbit-logs-dcr-output.json \
-n fluentbit-k8s-logs-dcr)

immutable_id=$(echo $dcr_response | jq -r '.immutableId')
dcr_resource_id=$(echo $dcr_response | jq -r '.id')
stream_name=$(echo $dcr_response | jq -r '.dataFlows[0].streams[0]')

echo "Creating log analytics workspace identity..."
identity=$(az identity create --resource-group $RESOURCE_GROUP --name $IDENITY_NAME --location $LOCATION)

az role assignment create --role "Monitoring Metrics Publisher" --assignee-principal-type ServicePrincipal --assignee $IDENITY_NAME --scope $dcr_resource_id

echo "Successfully created all the infrastructure."
echo "Endpoint URI: $data_collection_endpoint"
echo "Dcr immutable id $immutable_id"
echo "Stream name: $stream_name"

echo "Azure managed identity details:"
echo "client id: $(echo $identity | jq -r '.clientId')"
echo "tenant id: $(echo $identity | jq -r '.tenantId')"
