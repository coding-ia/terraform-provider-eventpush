---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "eventpush_aws_sns_publish_message Resource - terraform-provider-eventpush"
subcategory: ""
description: |-
  Send a message to an AWS SNS Topic.
---

# eventpush_aws_sns_publish_message (Resource)

Send a message to an AWS SNS Topic.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `message_body` (String) The message to send.
- `topic_arn` (String) The topic you want to publish to.

### Optional

- `create_only` (Boolean) When enabled, forces resource to be replaced on update.
- `kms_signature` (Block List) (see [below for nested schema](#nestedblock--kms_signature))

### Read-Only

- `event_id` (String) Generated ID for resource tracking.
- `md5_of_message_body` (String) The MD5 of the message body.

<a id="nestedblock--kms_signature"></a>
### Nested Schema for `kms_signature`

Required:

- `kms_key_id` (String) The ID of the AWS KMS key.

Optional:

- `algorithm` (String) The KMS signature algorithm.
- `message_attribute` (String) Message attribute name to add signature value.
