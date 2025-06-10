package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &EventPushProvider{}

type EventPushProvider struct {
	version string
}

func (e *EventPushProvider) Metadata(ctx context.Context, request provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "eventpush"
	response.Version = e.version
}

func (e *EventPushProvider) Schema(ctx context.Context, request provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "The automation Terraform provider contains various resources used to assist in automation.",
		Attributes: map[string]schema.Attribute{
			"profile": schema.StringAttribute{
				Description: "The profile for API operations. If not set, the default profile for aws configuration will be used.",
				Optional:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region in AWS where actions will take place.",
				Optional:    true,
			},
		},
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
