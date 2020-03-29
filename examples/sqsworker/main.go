package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/aws/aws-lambda-go/events" // does not really use lambda, just the S3 event struct
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	jobqueue "github.com/dirkaholic/kyoo"
	"github.com/dirkaholic/kyoo/job"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

const (
	EnvironmentLocal = "local"
)

func main() {
	log.SetOutput(os.Stdout)

	var cfg Config
	err := envconfig.Process("s3fileprocessor", &cfg)
	if err != nil {
		log.Fatal(err.Error())
	}

	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}

	sess := constructAWSSession(&cfg)

	sqsSvc := sqs.New(sess)
	s3Svc := s3.New(sess)

	c := Consumer{
		SQSSvc:      sqsSvc,
		S3Svc:       s3Svc,
		QueueName:   cfg.Queue,
		WorkerPool:  runtime.NumCPU() * 2,
		MaxMessages: 10, // 10 is current maximum value for MaxNumberOfMessages
	}

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		c.Stop()
		os.Exit(1)
	}()

	cErr := c.Consume()

	if cErr != nil {
		log.Fatalln("Error while running sqs consumer: ", cErr)
	}
}

// constructAWSSession constructs an AWS session from the config parameters given
func constructAWSSession(cfg *Config) *session.Session {
	defaultResolver := endpoints.DefaultResolver()

	awsRegion := cfg.Region
	if cfg.Environment == EnvironmentLocal {
		awsRegion = "us-east-1"
	}

	resolverFn := func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		if cfg.Environment != EnvironmentLocal {
			return defaultResolver.EndpointFor(service, region, optFns...)
		}

		var svcURL string
		switch service {
		case endpoints.SqsServiceID:
			svcURL = "http://localhost:4576"
		case endpoints.S3ServiceID:
			svcURL = "http://localhost:4572"
		default:
			return defaultResolver.EndpointFor(service, region, optFns...)
		}

		log.Debugf("Retrieving local %s endpoint ...", service)
		return endpoints.ResolvedEndpoint{
			URL:           svcURL,
			SigningRegion: awsRegion,
		}, nil
	}

	credVerboseErrors := true

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region:                        aws.String(awsRegion),
			EndpointResolver:              endpoints.ResolverFunc(resolverFn),
			CredentialsChainVerboseErrors: &credVerboseErrors,
		},
	}))

	if cfg.DebugAWS {
		sess.Config.MergeIn(&aws.Config{
			LogLevel: aws.LogLevel(aws.LogDebug),
		})
	}

	return sess
}

// Config represents configuration parameter read from the OS environment
type Config struct {
	Debug       bool   `default:"false"`
	DebugAWS    bool   `default:"false"`
	Environment string `default:"prod"`
	Region      string `default:"eu-central-1"`
	Queue       string `required:"true"`
}

// Consumer defines SQS queue consumer
type Consumer struct {
	SQSSvc      sqsiface.SQSAPI
	S3Svc       s3iface.S3API
	MaxMessages int64
	WorkerPool  int
	QueueName   string
	queueURL    *string
	queue       *jobqueue.JobQueue
}

// Consume starts consuming messages from SQS
func (c *Consumer) Consume() error {
	err := c.resolveQueueURL()

	if err != nil {
		return err
	}

	// start job queue and pool of workers
	c.queue = jobqueue.NewJobQueue(c.WorkerPool)
	c.queue.Start()

	log.Info("Start consuming messages with maxMessages set to: ", c.MaxMessages)
	for !c.queue.Stopped() {
		output, err := c.retrieveMessages(c.MaxMessages)
		if err != nil {
			log.Errorf("Error while retrieving messages: %s", err)
			continue
		}

		log.Debugf("Retrieved %d messages from queue", len(output.Messages))
		for _, message := range output.Messages {
			m := message // local copy of the mesage, avoid data race
			c.queue.Submit(&job.FuncExecutorJob{Func: func() error {
				c.processSQSMessage(m)
				return nil
			}})
		}
	}

	return nil
}

// Stop stops all workers and the queue gracefully
func (c *Consumer) Stop() {
	log.Debugln("Stopping consumer ...")
	c.queue.Stop()
}

// ConsumeSingleMessage consumes a single message from SQS
func (c *Consumer) ConsumeSingleMessage() error {
	err := c.resolveQueueURL()

	if err != nil {
		return err
	}

	output, err := c.retrieveMessages(1)
	if err != nil {
		log.Errorf("Error while retrieving messages: %s", err)
		return err
	}

	if len(output.Messages) > 0 {
		c.processSQSMessage(output.Messages[0])
	}

	return nil
}

func (c *Consumer) resolveQueueURL() error {
	qURL, err := c.SQSSvc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(c.QueueName),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == sqs.ErrCodeQueueDoesNotExist {
			return fmt.Errorf("Unable to find queue %q", c.QueueName)
		}
		return fmt.Errorf("Unable to get URL for queue %q, %v", c.QueueName, err)
	}

	log.Debugf("Resolved queue URL output: %v", qURL)
	c.queueURL = qURL.QueueUrl

	return nil
}

func (c *Consumer) retrieveMessages(maxMessages int64) (*sqs.ReceiveMessageOutput, error) {
	result, err := c.SQSSvc.ReceiveMessage(&sqs.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		QueueUrl:            c.queueURL,
		MaxNumberOfMessages: aws.Int64(maxMessages),
		VisibilityTimeout:   aws.Int64(60), // 60 seconds
		WaitTimeSeconds:     aws.Int64(20), // current maximum in order to use longpolling
	})

	if err != nil {
		return &sqs.ReceiveMessageOutput{}, err
	}

	return result, nil
}

func (c *Consumer) deleteMessage(message *sqs.Message) (*sqs.DeleteMessageOutput, error) {
	resultDelete, err := c.SQSSvc.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      c.queueURL,
		ReceiptHandle: message.ReceiptHandle,
	})

	if err != nil {
		return &sqs.DeleteMessageOutput{}, err
	}

	log.Debugf("Message %s deleted", *message.MessageId)
	return resultDelete, nil
}

func (c *Consumer) processSQSMessage(m *sqs.Message) {
	log.Debugf("Getting body: %s", *m.Body)

	var snsMsg sns.PublishInput

	snsErr := json.Unmarshal([]byte(*m.Body), &snsMsg)
	if snsErr != nil {
		log.Errorf("Error while unmarshaling SNS message: %s", snsErr)
	}

	log.Debugln("SNS message:", *snsMsg.Message)
	var s3Event events.S3Event

	err := json.Unmarshal([]byte(*snsMsg.Message), &s3Event)
	if err != nil {
		log.Errorf("Error while unmarshaling S3 event: %s", err)
	}

	for _, record := range s3Event.Records {
		log.Debugf("Processing S3 event from bucket: %s", record.S3.Bucket.Name)
		log.Debugf("Processing S3 object: %s", record.S3.Object.Key)

		// Do further file processing here
	}

	// Message consumed
	if _, err := c.deleteMessage(m); err != nil {
		log.Errorf("Error while deleting message: %s", err)
	}
}
