#!/bin/bash

RESOURCE_GROUP=fluentbit-logs-rg
WORKSPACE_NAME=fluentbit-logs-k8s
TABLE_NAME=fluentbit_logs_k8s
DATA_COLLECTION_ENDPOINT=fluentbit_logs_endpoint
LOCATION=westeurope

#TODO check identity type
az monitor log-analytics-workspace create --resource-group $RESOURCE_GROUP --name $WORKSPACE_NAME --location $LOCATION --identity-type UserAssigned --sku Free

logs_analytics_table=$(az monitor log-analytics workspace table create --workspace-name $WORKSPACE_NAME --resource-group $RESOURCE_GROUP --name "${TABLE_NAME}_CL" \
--columns TimeGenerated=datetime kubernetes_pod_name=string kubernetes_pod_id=string kubernetes_namespace_name=string kubernetes_host=string \
kubernetes_docker_id=string kubernetes_container_name=string kubernetes_container_image=string kubernetes_container_hash=string log=string stream=string \
--plan Basic)

monitor_endpoint=$(az monitor data-collection endpoint create --name $DATA_COLLECTION_ENDPOINT --resource-group $RESOURCE_GROUP --public-network-access Enabled --location $LOCATION)

#TODO extract correct values
./scripts/create-dcr-template/generate-dcr.sh <monitor-endpoint> <workspace-resource-id> $TABLE_NAME

az monitor data-collection rule create \
--location $LOCATION \
--resource-group $RESOURCE_GROUP \
--endpoint-id <monitor-endpoint> \
--rule-file ./scripts/create-dcr-template/fluentbit-logs-dcr-output.json \
-n fluentbit-k8s-logs-dcr