package provider

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &AWSSQSSendMessageResource{}
var _ resource.ResourceWithConfigure = &AWSSQSSendMessageResource{}

type AWSSQSSendMessageResource struct {
	AWSClient *AWSClient
}

type AWSSQSSendMessageResourceModel struct {
	CreateOnly       types.Bool                   `tfsdk:"create_only"`
	DelaySeconds     types.Int32                  `tfsdk:"delay_seconds"`
	EventId          types.String                 `tfsdk:"event_id"`
	KMSSignature     []KMSSignatureAttributeModel `tfsdk:"kms_signature"`
	MD5OfMessageBody types.String                 `tfsdk:"md5_of_message_body"`
	MessageBody      types.String                 `tfsdk:"message_body"`
	QueueUrl         types.String                 `tfsdk:"queue_url"`
}

type KMSSignatureAttributeModel struct {
	KMSKeyID         types.String `tfsdk:"kms_key_id"`
	MessageAttribute types.String `tfsdk:"message_attribute"`
	Algorithm        types.String `tfsdk:"algorithm"`
}

func newAWSSQSSendMessageResource() resource.Resource {
	return &AWSSQSSendMessageResource{}
}

func (r *AWSSQSSendMessageResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

	sqsClient := sqs.NewFromConfig(cfg)
	kmsClient := kms.NewFromConfig(cfg)

	r.AWSClient = &AWSClient{
		SQSClient: sqsClient,
		KMSClient: kmsClient,
		Region:    providerMeta.AWSConfigOptions.Region,
	}
}

func (r *AWSSQSSendMessageResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_aws_sqs_send_message"
}

func (r *AWSSQSSendMessageResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Send a message to an AWS SQS Queue.",
		Attributes: map[string]schema.Attribute{
			"create_only": schema.BoolAttribute{
				Description: "When enabled, forces resource to be replaced on update.",
				Optional:    true,
			},
			"delay_seconds": schema.Int32Attribute{
				Description: "The length of time, in seconds, for which to delay a specific message.",
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
			"queue_url": schema.StringAttribute{
				Description: "The URL of the Amazon SQS queue which a message is sent.",
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

func (r *AWSSQSSendMessageResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data AWSSQSSendMessageResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	err := sendMessage(ctx, r.AWSClient, &data, "create")
	if err != nil {
		response.Diagnostics.AddError("Error sending message to SQS queue.", err.Error())
		return
	}

	// Set event ID only in creation lifecycle
	data.EventId = types.StringValue(uuid.New().String())

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
	var plan, state AWSSQSSendMessageResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)

	if response.Diagnostics.HasError() {
		return
	}

	planMessageBodyMD5 := createMD5OfMessageBody(plan.MessageBody.ValueString())
	stateMessageBodyMD5 := createMD5OfMessageBody(state.MessageBody.ValueString())

	if planMessageBodyMD5 != stateMessageBodyMD5 {
		err := sendMessage(ctx, r.AWSClient, &plan, "update")
		if err != nil {
			response.Diagnostics.AddError("Error sending message to SQS queue.", err.Error())
			return
		}
	} else {
		plan.MD5OfMessageBody = types.StringValue(planMessageBodyMD5)
	}

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *AWSSQSSendMessageResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data AWSSQSSendMessageResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	if !data.CreateOnly.ValueBool() {
		err := sendMessage(ctx, r.AWSClient, &data, "delete")
		if err != nil {
			response.Diagnostics.AddError("Error sending message to SQS queue.", err.Error())
			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func sendMessage(ctx context.Context, meta *AWSClient, data *AWSSQSSendMessageResourceModel, lifeCycle string) error {
	messageAttributes := make(map[string]sqstypes.MessageAttributeValue)
	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(data.QueueUrl.ValueString()),
		MessageBody: aws.String(data.MessageBody.ValueString()),
	}

	if !data.DelaySeconds.IsNull() {
		input.DelaySeconds = data.DelaySeconds.ValueInt32()
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

		messageAttributes[attrName] = sqstypes.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(signature),
		}
	}

	messageAttributes["X-LifeCycle-Event"] = sqstypes.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String(lifeCycle),
	}

	input.MessageAttributes = messageAttributes
	output, err := meta.SQSClient.SendMessage(ctx, input)

	if err != nil {
		return err
	}

	data.MD5OfMessageBody = types.StringPointerValue(output.MD5OfMessageBody)

	return nil
}

func createMD5OfMessageBody(input string) string {
	hash := md5.Sum([]byte(input))
	hashString := hex.EncodeToString(hash[:])
	return hashString
}
