package provider

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &AWSSNSPublishMessageResource{}
var _ resource.ResourceWithConfigure = &AWSSNSPublishMessageResource{}

type AWSSNSPublishMessageResource struct {
	AWSClient *AWSClient
}

type AWSSNSPublishMessageResourceModel struct {
	CreateOnly       types.Bool                   `tfsdk:"create_only"`
	EventId          types.String                 `tfsdk:"event_id"`
	KMSSignature     []KMSSignatureAttributeModel `tfsdk:"kms_signature"`
	MD5OfMessageBody types.String                 `tfsdk:"md5_of_message_body"`
	MessageBody      types.String                 `tfsdk:"message_body"`
	TopicARN         types.String                 `tfsdk:"topic_arn"`
}

func newAWSSNSPublishMessageResource() resource.Resource {
	return &AWSSNSPublishMessageResource{}
}

func (r *AWSSNSPublishMessageResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	providerMeta := request.ProviderData.(Meta)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		response.Diagnostics.AddError("unable to load SDK config", err.Error())
		return
	}

	if providerMeta.AWSConfigOptions.Region != "" {
		cfg.Region = providerMeta.AWSConfigOptions.Region
	}

	snsClient := sns.NewFromConfig(cfg)
	kmsClient := kms.NewFromConfig(cfg)

	r.AWSClient = &AWSClient{
		SNSClient: snsClient,
		KMSClient: kmsClient,
		Region:    providerMeta.AWSConfigOptions.Region,
	}
}

func (r *AWSSNSPublishMessageResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_aws_sns_publish_message"
}

func (r *AWSSNSPublishMessageResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Send a message to an AWS SNS Topic.",
		Attributes: map[string]schema.Attribute{
			"create_only": schema.BoolAttribute{
				Description: "When enabled, forces resource to be replaced on update.",
				Optional:    true,
			},
			"event_id": schema.StringAttribute{
				Description: "Generated ID for resource tracking.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"md5_of_message_body": schema.StringAttribute{
				Description: "The MD5 of the message body.",
				Computed:    true,
			},
			"message_body": schema.StringAttribute{
				Description: "The message to send.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(replaceIfCreateOnlySet, "Forces replacement of resource.", "Forces replacement of resource."),
				},
			},
			"topic_arn": schema.StringAttribute{
				Description: "The topic you want to publish to.",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"kms_signature": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"kms_key_id": schema.StringAttribute{
							Description: "The ID of the AWS KMS key.",
							Required:    true,
						},
						"message_attribute": schema.StringAttribute{
							Description: "Message attribute name to add signature value.",
							Optional:    true,
						},
						"algorithm": schema.StringAttribute{
							Description: "The KMS signature algorithm.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.OneOfCaseInsensitive(
									[]string{
										string(kmstypes.SigningAlgorithmSpecRsassaPkcs1V15Sha256),
									}...,
								),
							},
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
		},
	}
}

func (r *AWSSNSPublishMessageResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data AWSSNSPublishMessageResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	err := publishMessage(ctx, r.AWSClient, &data, "create")
	if err != nil {
		response.Diagnostics.AddError("Error sending message to SNS topic.", err.Error())
		return
	}

	data.EventId = types.StringValue(uuid.New().String())
	data.MD5OfMessageBody = types.StringValue(createMD5OfMessageBody(data.MessageBody.ValueString()))

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *AWSSNSPublishMessageResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data AWSSNSPublishMessageResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *AWSSNSPublishMessageResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state AWSSNSPublishMessageResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)

	if response.Diagnostics.HasError() {
		return
	}

	planMessageBodyMD5 := createMD5OfMessageBody(plan.MessageBody.ValueString())
	stateMessageBodyMD5 := createMD5OfMessageBody(state.MessageBody.ValueString())

	if planMessageBodyMD5 != stateMessageBodyMD5 {
		err := publishMessage(ctx, r.AWSClient, &plan, "update")
		if err != nil {
			response.Diagnostics.AddError("Error sending message to SNS topic.", err.Error())
			return
		}
	}
	plan.MD5OfMessageBody = types.StringValue(planMessageBodyMD5)

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *AWSSNSPublishMessageResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data AWSSNSPublishMessageResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	if !data.CreateOnly.ValueBool() {
		err := publishMessage(ctx, r.AWSClient, &data, "delete")
		if err != nil {
			response.Diagnostics.AddError("Error sending message to SNS topic.", err.Error())
			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)

}

func publishMessage(ctx context.Context, meta *AWSClient, data *AWSSNSPublishMessageResourceModel, lifeCycle string) error {
	messageAttributes := make(map[string]snstypes.MessageAttributeValue)
	input := &sns.PublishInput{
		TopicArn: aws.String(data.TopicARN.ValueString()),
		Message:  aws.String(data.MessageBody.ValueString()),
	}

	if data.KMSSignature != nil {
		kmsBlock := data.KMSSignature[0]

		attrName := "X-KMS-Signature"
		if !kmsBlock.MessageAttribute.IsNull() {
			attrName = kmsBlock.MessageAttribute.String()
		}

		algorithm := "RSASSA_PKCS1_V1_5_SHA_256"
		if !kmsBlock.Algorithm.IsNull() {
			algorithm = kmsBlock.Algorithm.String()
		}

		signature, err := signMessageBodyWithKMS(ctx, meta.KMSClient, algorithm, kmsBlock.KMSKeyID.ValueString(), data.MessageBody.ValueString())
		if err != nil {
			return err
		}

		messageAttributes[attrName] = snstypes.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(signature),
		}
	}

	messageAttributes["X-LifeCycle-Event"] = snstypes.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String(lifeCycle),
	}

	input.MessageAttributes = messageAttributes
	_, err := meta.SNSClient.Publish(ctx, input)
	return err
}

func replaceIfCreateOnlySet(ctx context.Context, request planmodifier.StringRequest, response *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	var createOnly types.Bool

	response.Diagnostics.Append(request.State.GetAttribute(ctx, path.Root("create_only"), &createOnly)...)

	if response.Diagnostics.HasError() {
		return
	}

	if !createOnly.IsNull() {
		if createOnly.ValueBool() {
			response.RequiresReplace = true
		}
	}
}
