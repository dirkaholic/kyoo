#!/bin/bash

# create S3 bucket, topic adn queue for images
awslocal s3 mb s3://local-kyoo-bucket
awslocal sns create-topic --name kyoo-images-topic
awslocal sqs create-queue --queue-name kyoo-images-queue

# allow SNS to send message to SQS queue
jq -c '. | { Policy: @text }' /config/sqs-permission.json > /tmp/sqs.json
awslocal sqs set-queue-attributes --queue-url "http://localhost:4576/queue/kyoo-images-queue" --attributes file://tmp/sqs.json

# subscribe SQS to SNS topic
awslocal sns subscribe --topic-arn "arn:aws:sns:us-east-1:000000000000:kyoo-images-topic" --protocol "sqs" --notification-endpoint "arn:aws:sqs:us-east-1:000000000000:kyoo-images-queue"

# allow S3 to publish to SNS
awslocal sns set-topic-attributes --topic-arn "arn:aws:sns:us-east-1:000000000000:kyoo-images-topic" --attribute-name Policy --attribute-value file://config/sns-permission.json

# add a notification to SNS for S3 on creating new object
awslocal s3api put-bucket-notification-configuration --bucket local-kyoo-bucket --notification-configuration file://config/s3-notification.json
