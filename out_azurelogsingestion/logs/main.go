package logs

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
)

type AzureLogsClient interface {
	Upload(ctx context.Context,
		dcrImmutableId string,
		streamName string,
		logs []byte,
		options *azlogs.UploadOptions) (azlogs.UploadResponse, error)
}
