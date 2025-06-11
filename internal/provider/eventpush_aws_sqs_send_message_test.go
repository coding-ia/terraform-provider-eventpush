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
}
`

	config2 := `
resource "eventpush_aws_sqs_send_message" "test" {
  message_body = "test message 2"
  queue_url    = "https://sqs.us-east-2.amazonaws.com/242306084486/TestQueue"
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

func TestAccEventPushSQSSendMessage_Delete(t *testing.T) {
	config1 := `
resource "eventpush_aws_sqs_send_message" "test" {
  message_body = "test message 2"
  queue_url    = "https://sqs.us-east-2.amazonaws.com/242306084486/TestQueue"
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
				Config:  config1,
				Destroy: true,
			},
		},
	})
}
