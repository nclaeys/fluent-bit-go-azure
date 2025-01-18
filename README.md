# fluent-bit-go-azure
Fluentbit output plugin in go to write to Azure logs ingestion API from kubernetes.
The default logs ingestion output plugin does not support workload identity and relies on client id and client secret for interacting with the API.

However, in my projects I cannot use this for 2 reasons:
- I do not have access to the Entra ID tenant using Iac to register applications. For this I manually need to create tickets with support. I prefer to use user managed identity and give them access to Azure resources.
- I think it is a bad practice to rely on a static client_secret for production applications. These secrets are not rotated and once leaked can be used forever.

For exactly this reason, workload identity on kubernetes exists to get temporary credentials to access azure resources.

## Context
A year ago, Azure introduced a new API for processing all logs and metrics.
More details can be found [here](https://learn.microsoft.com/en-us/azure/azure-monitor/logs/logs-ingestion-api-overview).

This is a replacement of the data collector API, which existed before.
One of the biggest drivers for using this new endpoint is that it supports `Basic tables`, which are around 5 times cheaper than analytics tables.

## Setting up the logs ingestion API

1. Create a log analytics table with a schema. There is a PR to do this in terraform, for now you can run the following command using the Azure CLI:
```bash
az monitor log-analytics workspace table create --workspace-name <workspace-name> --resource-group <resource-group> --name <table-name>_CL \
--columns TimeGenerated=datetime kubernetes_pod_name=string kubernetes_pod_id=string kubernetes_namespace_name=string kubernetes_host=string \
kubernetes_docker_id=string kubernetes_container_name=string kubernetes_container_image=string kubernetes_container_hash=string log=string stream=string \
--plan Basic
```

2. Create a data collection endpoint. This is a simple endpoint that you can use to send data to the logs ingestion API. This is needed to send our custom json data.
```bash
az monitor data-collection endpoint create --name <name> --resource-group <rg> --public-network-access Enabled --location westeurope
```

Add permission: Monitor metrics publisher to the sp that is used to send data to the endpoint.

Note: if you do not want public access, you can change the network-access to use the `disabled` or `securedByPerimeter` setting.

3. Create a data collection rule. This is required to specify the data format used as well as link our endpoint to our log analytics table.
   I included a template to generate a dcr for using fluentbit data in 'scripts/create-dcr-template'. For more details about the dcr see the [Azure documentation](https://learn.microsoft.com/en-us/azure/azure-monitor/essentials/data-collection-rule-create-edit?tabs=cli).
   You can generate the correct template using the following command:
```bash
./scripts/create-dcr-template/generate-dcr.sh <data-collection-endpoint-id> <workspace-resource-id> <logs-table-name>
```

Once the template is generated, you can create the dcr using the following command:
```bash
az monitor data-collection rule create \
--location <location> \
--resource-group <rg> \
--endpoint-id <endpoint-id> \
--rule-file ./scripts/create-dcr-template/fluentbit-logs-dcr-output.json \
-n fluentbit-k8s-logs-dcr

```

## Testing the endpoint

If you want to test the endpoint locally first, there is a simple script that writes some dummy data. 
To use it, you need to create an AAD application and give it the `Monitoring Metrics Publisher` role on the data collection endpoint.
Next, create a .env file in the root repository filling in the following values:
```bash
client_secret=
client_id=
tenant_id=
endpoint=https://xxx.ingest.monitor.azure.com
dcr_immutable_id=dcr-xxxxxx
stream_name=
```

## Packaging the plugin

