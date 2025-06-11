package provider

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &EventPushProvider{}

type EventPushProvider struct {
	version string
}

type Meta struct {
	SQSClient *sqs.Client
	KMSClient *kms.Client
}

func (e *EventPushProvider) Metadata(ctx context.Context, request provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "eventpush"
	response.Version = e.version
}

func (e *EventPushProvider) Schema(ctx context.Context, request provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "The Event Push provider contains resource used to send messages to various services.",
	}
}

func (e *EventPushProvider) Configure(ctx context.Context, request provider.ConfigureRequest, response *provider.ConfigureResponse) {

}

func (e *EventPushProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (e *EventPushProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newAWSSQSSendMessageResource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &EventPushProvider{
			version: version,
		}
	}
}

func signMessageBodyWithKMS(ctx context.Context, kmsClient *kms.Client, algorithm, keyID, message string) (string, error) {
	digest := sha256.Sum256([]byte(message))

	output, err := kmsClient.Sign(ctx, &kms.SignInput{
		KeyId:            aws.String(keyID),
		Message:          digest[:],
		MessageType:      kmstypes.MessageTypeDigest,
		SigningAlgorithm: kmstypes.SigningAlgorithmSpec(algorithm),
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	return base64.StdEncoding.EncodeToString(output.Signature), nil
}
