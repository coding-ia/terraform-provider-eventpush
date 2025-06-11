package provider

import (
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"testing"
)

func TestAccEventPushSQSSendMessage_Simple(t *testing.T) {
	config1 := `
resource "eventpush_aws_sqs_send_message" "test" {
  message_body = "test message 2"
  queue_url    = "https://sqs.us-east-2.amazonaws.com/242306084486/TestQueue"

  kms_signature {
    kms_key_id = "arn:aws:kms:us-east-2:242306084486:key/9834cc70-67b2-446b-b921-34feb2c33406"
  }
}
`

	config2 := `
resource "eventpush_aws_sqs_send_message" "test" {
  message_body = "test message 1"
  queue_url    = "https://sqs.us-east-2.amazonaws.com/242306084486/TestQueue"

  kms_signature {
    kms_key_id = "arn:aws:kms:us-east-2:242306084486:key/9834cc70-67b2-446b-b921-34feb2c33406"
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() {},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("eventpush_aws_sqs_send_message.test", "queue_url", "https://sqs.us-east-2.amazonaws.com/242306084486/TestQueue"),
				),
			},
			{
				RefreshState: true,
			},
			{
				Config: config2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("eventpush_aws_sqs_send_message.test", "queue_url", "https://sqs.us-east-2.amazonaws.com/242306084486/TestQueue"),
				),
			},
		},
	})
}
