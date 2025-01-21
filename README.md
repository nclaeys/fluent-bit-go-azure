# fluent-bit-go-azure

Fluentbit output plugin, written in go, to send logs to the Azure logs ingestion API.

The [default logs ingestion plugin](https://docs.fluentbit.io/manual/pipeline/outputs/azure_logs_ingestion) for fluentbit does not work for me because:
- At many clients I have no access to Microsoft Entra ID directly from Terraform. 
  This means that I cannot create AAD applications myself but need to create support tickets in order to register AAD applications, which breaks my IAC code. 
  For these reasons, I prefer to depend on user managed identities instead of AAD applications.
- It is a bad practice to rely on a static `client_secret` values for production applications as is required by the default plugin. 
  As these secrets are not rotated, they can be used forever when leaked. 
  I want to use temporary access credentials as is provided using [Azure workload identity](https://azure.github.io/azure-workload-identity/docs/) on k8s.

Because of both reasons, I created this plugin such that I can use workload identity with managed identities to send my logs to the logs ingestion endpoint.
I wrote it in Golang as I am terrible at C and fluentbit supports output plugins written in Golang as described [here](https://docs.fluentbit.io/manual/development/golang-output-plugins).

## Prerequisites to get started

- Install the [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli)
- Having access to an Azure subscription

## Getting started

### Infrastructure all in one

Make sure Azure CLI is installed and you are logged in.
The easiest way to get started to use the `all-in-one` script, which will create all the necessary Azure resources.
All resources a dedicated resource group, take a look at the variables on top of the script before running it:

```bash
./scripts/all-in-one.sh
```
For a detailed description of what the script does, see the [details](#detailed-explanation-of-azure-resources-required) section.

### Deploy fluentbit on Kubernetes

The list of Docker images can be found [here](https://hub.docker.com/repository/docker/nilli9990/fluentbit-go-azure-logs-ingestion/general).

In the kubernetes folder, you will find a standard example for deploying fluentbit using this plugin on kubernetes.
1. Replace the `AZURE_CLIENT_ID` and `AZURE_TENANT_ID` in the `kubernetes/fluentbit-deployment.yaml` file with the values of the user managed identity you created.
2. Fill in the necessary fluentbit configuration in the `kubernetes/configmap.yaml` file. 
Here you must fill in the `Endpoint`, `DcrImmutableId` and `StreamName` fields with their respective values.
For more details on how to configure fluentbit, see the [fluentbit documentation](https://docs.fluentbit.io/manual/administration/configuring-fluent-bit/yaml).

## Detailed explanation of Azure resources required
Alternatively, you can follow the different steps below to alter the individual steps.

1. Create a custom fluentbit table in your log analytics workspace. 
   There exists a PR to do this in terraform, but it is not merged yet. for now you can do this by running the following command using the Azure CLI:

```bash
az monitor log-analytics workspace table create --workspace-name <workspace-name> --resource-group <resource-group> --name <table-name>_CL \
--columns TimeGenerated=datetime kubernetes_pod_name=string kubernetes_pod_id=string kubernetes_namespace_name=string kubernetes_host=string \
kubernetes_docker_id=string kubernetes_container_name=string kubernetes_container_image=string kubernetes_container_hash=string log=string stream=string \
--plan Basic
```

2. Create a data collection endpoint. This is the endpoint that will be used to send your fluentbit logs. This is required when using custom json data.

```bash
az monitor data-collection endpoint create --name <name> --resource-group <rg> --public-network-access Enabled --location <location>
```

Note: if you do not want public access, you can change the network-access to use the `disabled` or `securedByPerimeter` setting.

3. Create a data collection rule. This is required as it specifes the data format used as well as links the source (your endpoint) to the destination (your log analytics table).
   This repo contains a template to generate a basic dcr for fluentbit data in the `scripts/create_dcr` folder.
   For more details about the dcr specification see the [Azure documentation](https://learn.microsoft.com/en-us/azure/azure-monitor/essentials/data-collection-rule-create-edit?tabs=cli).
   You can generate a valid output template for you using:

```bash
./scripts/create-dcr-template/generate-dcr.sh <data-collection-endpoint-uri> <workspace-resource-id> <logs-table-name>
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

4. Create a user managed identity for your fluentbit pods by running:
```bash
identity=$(az identity create --resource-group $RESOURCE_GROUP --name $IDENITY_NAME --location $LOCATION)
```

From this you will need the client_id and tenant_id such that your fluentbit pods can use Azure workload identity to authenticate with the Azure API.

5. Create a role assignment for the user managed identity to be able to send logs to the logs ingestion API.
```bash
az role assignment create --role "Monitoring Metrics Publisher" --assignee-principal-type ServicePrincipal --assignee $IDENITY_NAME --scope $data_collection_endpoint_id
```

## Testing your logs endpoint
For this test scripts, you need to have Golang installed [Go](https://go.dev/doc/install)

Note: Make sure your user, AAD application or service principal, which you will use to send logs, has the role `Monitor metrics publisher` attached with scope the data collection endpoint.
For this you cannot use the managed identity just created, as that only works on Azure resources.

If you want to test the endpoint locally first, there is a simple script that writes some dummy data.
Next, create a .env file in the root repository filling in the following values:
```bash
client_secret=
client_id=
tenant_id=
endpoint=https://xxx.ingest.monitor.azure.com
dcr_immutable_id=dcr-xxxxxx
stream_name=
```

## Background: changes in logging on Azure
A year ago, Azure introduced a new API for processing logs and metrics integrated in Azure monitor, called the logs ingestion API.
More details can be found [here](https://learn.microsoft.com/en-us/azure/azure-monitor/logs/logs-ingestion-api-overview).

This API is a replacement of the legacy data collector API.
The biggest driver why I want to use this new endpoint is that it supports `Basic tables`, which are around 5 times cheaper than analytics tables.
