package provider

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &AWSSQSSendMessageResource{}
var _ resource.ResourceWithConfigure = &AWSSQSSendMessageResource{}

type AWSSQSSendMessageResource struct {
	sqsClient *sqs.Client
}

type AWSSQSSendMessageResourceModel struct {
	DelaySeconds types.Int32                 `tfsdk:"delay_seconds"`
	KMSSignature *KMSSignatureAttributeModel `tfsdk:"kms_signature"`
	MessageBody  types.String                `tfsdk:"message_body"`
	MessageId    types.String                `tfsdk:"message_id"`
	QueueUrl     types.String                `tfsdk:"queue_url"`
}

type KMSSignatureAttributeModel struct {
	KMSKeyID         types.String `tfsdk:"kms_key_id"`
	MessageAttribute types.String `tfsdk:"message_attribute"`
}

func newAWSSQSSendMessageResource() resource.Resource {
	return &AWSSQSSendMessageResource{}
}

func (r *AWSSQSSendMessageResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		response.Diagnostics.AddError("unable to load SDK config", err.Error())
		return
	}

	r.sqsClient = sqs.NewFromConfig(cfg)
}

func (r *AWSSQSSendMessageResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_aws_sqs_send_message"
}

func (r *AWSSQSSendMessageResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Send a message to an AWS SQS Queue.",
		Attributes: map[string]schema.Attribute{
			"delay_seconds": schema.Int32Attribute{
				Description: "The length of time, in seconds, for which to delay a specific message.",
				Optional:    true,
			},
			"message_body": schema.StringAttribute{
				Description: "The message to send.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"message_id": schema.StringAttribute{
				Description: "Message ID return by the queue.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"queue_url": schema.StringAttribute{
				Description: "The URL of the Amazon SQS queue which a message is sent.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"kms_signature": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"kms_key_id": schema.StringAttribute{
						Description: "The ID of the AWS KMS key.",
						Optional:    true,
					},
					"message_attribute": schema.StringAttribute{
						Description: "Message attribute name to add signature value.",
						Optional:    true,
					},
				},
			},
		},
	}
}

func (r *AWSSQSSendMessageResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data AWSSQSSendMessageResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(data.QueueUrl.ValueString()),
		MessageBody: aws.String(data.MessageBody.ValueString()),
	}

	if !data.DelaySeconds.IsNull() {
		input.DelaySeconds = data.DelaySeconds.ValueInt32()
	}

	output, err := r.sqsClient.SendMessage(ctx, input)

	if err != nil {
		response.Diagnostics.AddError("Error sending message to SQS queue.", err.Error())
		return
	}

	data.MessageId = types.StringPointerValue(output.MessageId)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *AWSSQSSendMessageResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data AWSSQSSendMessageResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *AWSSQSSendMessageResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {

}

func (r *AWSSQSSendMessageResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data AWSSQSSendMessageResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}
