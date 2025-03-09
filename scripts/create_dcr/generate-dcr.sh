#!/bin/bash
set -ex

cd "$(dirname "$0")"
echo "executing script from: $(pwd)"

# This script generates a DCR file from a template file by replacing the placeholders with the actual values.
DCR_TEMPLATE_FILE=fluentbit-logs-dcr-template.json
DATA_COLLECTION_ENDPOINT=$1
WORKSPACE_RESOURCE_ID=$2
OUTPUT_TABLE_NAME=$3

if [ -z "$DATA_COLLECTION_ENDPOINT" ] || [ -z "$WORKSPACE_RESOURCE_ID" ] || [ -z "$OUTPUT_TABLE_NAME" ]; then
    echo "Usage: $0 <data-collection-endpoint> <workspace-resource-id> <output-table-name>"
    exit 1
fi

DCR_FILE=fluentbit-logs-dcr-output.json
cp $DCR_TEMPLATE_FILE $DCR_FILE
sed -i "s|DATA_COLLECTION_ENDPOINT_ID|$DATA_COLLECTION_ENDPOINT|g" $DCR_FILE
sed -i "s|WORKSPACE_RESOURCE_ID|$WORKSPACE_RESOURCE_ID|g" $DCR_FILE
sed -i "s|TABLE_NAME|$OUTPUT_TABLE_NAME|g" $DCR_FILE
