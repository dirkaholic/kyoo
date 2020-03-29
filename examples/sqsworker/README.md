# Reference implementation for SQS worker using the kyoo library

## Content

* Installs [localstack](https://github.com/localstack/localstack) - a fully functional local AWS cloud stack
* Sets up some services and connects them
    * AWS S3 Bucket Notification to SQS queue via SNS topic

## Usage

### Start localstack and execute setup steps

```bash
docker-compose up -d
```

### Commands

Run SQS worker

```bash
export $(cat sqsworker.env | xargs) 
go build && ./sqsworker 
```

Upload image to S3 bucket

```bash
docker-compose run setup awslocal s3 cp /test/images/image.jpg s3://local-kyoo-bucket/image.jpg
```

Get messages from SQS queue

```bash
docker-compose run setup awslocal sqs receive-message --queue-url "http://localhost:4576/queue/kyoo-images-queue" --max-number-of-messages 10
```
