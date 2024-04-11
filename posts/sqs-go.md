

I feel liked to regarding AWS
After hearing horror story after horror story where $5,000 bills got emailed to developers at the end of the month I wrote off the idea of using any of their services.

Having not discovered their free tier ( hence why I feel lied to ) I was creating a personal project where I would have to watch 1,000's of objects, with each object being "completed" once in an api call. My worry was CPU overloading becoming an issue.

Having then stumbled across DreamsOfCode's video https://www.youtube.com/watch?v=O_0IGoOX6Dw , I realised how large the free tier's large size and figured I could solve my issues using this ( lambda functions to watch 100-200 at a time)



Initially I'll create four programs 
- Send SQS 
- Receive SQS + delete
- Recieve SQS in lambda 


GoLang is my language of choice here due to easy readability and experience.
// main.go
```go
package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/seal/sqs-hello/pkg/utils"
)

type SQSSendMessageAPI interface {
	GetQueueUrl(ctx context.Context,
		params *sqs.GetQueueUrlInput,
		optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)

	SendMessage(ctx context.Context,
		params *sqs.SendMessageInput,
		optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

func GetQueueURL(c context.Context, api SQSSendMessageAPI, input *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	return api.GetQueueUrl(c, input)
}
func SendMsg(c context.Context, api SQSSendMessageAPI, input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	return api.SendMessage(c, input)
}
func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}
	cfg.Region = "us-east-1"
	cfg.Credentials = credentials.NewStaticCredentialsProvider(utils.EnvVariable("ACCESS_KEY"), utils.EnvVariable("SECRET_KEY"), "")
	client := sqs.NewFromConfig(cfg)

	gQInput := &sqs.GetQueueUrlInput{
		QueueName: aws.String("q-2"),
	}

	result, err := GetQueueURL(context.TODO(), client, gQInput)
	if err != nil {
		fmt.Println("Got an error getting the queue URL:")
		fmt.Println(err)
		return
	}

	queueURL := result.QueueUrl

	sMInput := &sqs.SendMessageInput{
		DelaySeconds: 10,
		MessageAttributes: map[string]types.MessageAttributeValue{
			"Name": {
				DataType:    aws.String("String"),
				StringValue: aws.String("Bob"),
			},
			"Lambda": {
				DataType:    aws.String("String"),
				StringValue: aws.String("Physics symbol"),
			},
			"NumberType": {
				DataType:    aws.String("Number"),
				StringValue: aws.String("10"),
			},
		},
		MessageBody: aws.String("Message body regarding sqs message"),
		QueueUrl:    queueURL,
	}
	resp, err := SendMsg(context.TODO(), client, sMInput)
	if err != nil {
		fmt.Println("Got an error sending the message:")
		fmt.Println(err)
		return
	}
	fmt.Println("Sent message with ID: " + *resp.MessageId)

}
```
/pkg/utils.go
```go
package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func EnvVariable(key string) string {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err) // Should only fail if no file is available
	}
	return os.Getenv(key)
}

```

This was fairly simple, just sending a message using a modified version of the example code 
.env

```
ACCESS_KEY=XXX
SECRET_KEY=XXX
```

The desired output is something like this (UUID's will differ):
```
Sent message with ID: 1ea4827c-c316-4cf7-b265-4eca3f52b2e1
```


Sending was just as simple:
```go
package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/seal/sqs-hello/pkg/utils"
)

type SQSReceiveMessageAPI interface {
	GetQueueUrl(ctx context.Context,
		params *sqs.GetQueueUrlInput,
		optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)

	ReceiveMessage(ctx context.Context,
		params *sqs.ReceiveMessageInput,
		optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)

	DeleteMessage(ctx context.Context,
		params *sqs.DeleteMessageInput,
		optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

func GetQueueURL(c context.Context, api SQSReceiveMessageAPI, input *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	return api.GetQueueUrl(c, input)
}

func GetMessages(c context.Context, api SQSReceiveMessageAPI, input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	return api.ReceiveMessage(c, input)
}
func main() {

	queue := flag.String("q", "", "The name of the queue")
	timeout := flag.Int("t", 5, "How long, in seconds, that the message is hidden from others")

	delete := flag.Bool("d", false, "Do you want to delete the messages after reading")
	flag.Parse()
	if *queue == "" {
		fmt.Println("You must supply the name of a queue (-q QUEUE)")
		return
	}

	if *timeout < 0 {
		*timeout = 0
	}

	if *timeout > 12*60*60 {
		*timeout = 12 * 60 * 60
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}

	cfg.Region = *aws.String("us-east-1")

	cfg.Credentials = credentials.NewStaticCredentialsProvider(utils.EnvVariable("ACCESS_KEY"), utils.EnvVariable("SECRET_KEY"), "")

	client := sqs.NewFromConfig(cfg)
	gQInput := &sqs.GetQueueUrlInput{
		QueueName: queue,
	}

	urlResult, err := GetQueueURL(context.TODO(), client, gQInput)
	if err != nil {
		fmt.Println("Got an error getting the queue URL:")
		fmt.Println(err)
		return
	}

	queueURL := urlResult.QueueUrl
	gMInput := &sqs.ReceiveMessageInput{
		QueueUrl: queueURL,
		AttributeNames: []types.QueueAttributeName{
			"SentTimestamp",
		},
		MaxNumberOfMessages: 10,
		MessageAttributeNames: []string{
			"Name",
			"Lambda",
			"NumberType",
		},

		WaitTimeSeconds: int32(5),
	}
	for {

		msgResult, err := GetMessages(context.TODO(), client, gMInput)
		if err != nil {
			fmt.Println("Got an error receiving messages:")
			fmt.Println(err)
			return
		}

		if msgResult.Messages != nil {
			for _, message := range msgResult.Messages {
				fmt.Println("Message ID:     " + *message.MessageId)
				fmt.Println("Message Handle: " + *message.ReceiptHandle)
				fmt.Println("Timestamp" + message.Attributes["SentTimestamp"])
				// Accessing specific message attributes
				for attrName, attrValue := range message.MessageAttributes {
					fmt.Printf("Attribute Name: %s\n", attrName)
					fmt.Printf("Attribute Value: %s\n", *attrValue.StringValue)
					fmt.Printf("Attribute DataType: %s\n", *attrValue.DataType)
				}
				fmt.Println("Body:", *message.Body)
				if *delete {
					dMInput := &sqs.DeleteMessageInput{
						QueueUrl:      queueURL,
						ReceiptHandle: message.ReceiptHandle,
					}
					_, err = RemoveMessage(context.TODO(), client, dMInput)
					if err != nil {
						fmt.Println("Got an error deleting the message:")
						fmt.Println(err)
						return
					} else {
						fmt.Println("Deleted message with handler " + *message.ReceiptHandle)
					}
				}
			}
		} else {
			fmt.Println("No messages found, trying again in 10 seconds")
			time.Sleep(10 * time.Second)
		}
	}

}

func RemoveMessage(c context.Context, api SQSReceiveMessageAPI, input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	return api.DeleteMessage(c, input)
}

```

Running this via 
```
go run . -q="q-2" -d=true
```

We receive :
```
Message ID:     1ea4827c-c316-4cf7-b265-4eca3f52b2e1
Message Handle: AQEBXSZRwAN5a4h8D0oXpoQ9WItjbd7ZZGG66ZeKcIWjOMUxkG28Z6K9iG6WFa6giAM5w3Yt1v/qLhvO38gZNuwm6aV8g5CxIm6TsAEGNkmHWz2sgv/aiDiTUIahD5JbrfAIVPDFVGNMFth9FBL80wQDeWmxLg8Ha3+QMkB6wh8XC7Bps5AwkeLXjg2o4w/u4TlSxXdCkQttbTW2dKiCujTxzvQ5q/SYi5pnH95UD/3ef4n1W3KOCmTswQVZKrdRDyougdXD+9ZLSd1D85zUPTO1PGofscvgf8n7KSyoRepl/tjb6V5dI3u00ckclk90V+dFu0pMZBbg3/N9AfReOGBmJGcl6VsCleGbKeTI/8X84imWj53AlYc1DRM0SpBOGVr
Timestamp1694527359000
Attribute Name: Lambda
Attribute Value: Physics symbol
Attribute DataType: String
Attribute Name: Name
Attribute Value: Bob
Attribute DataType: String
Attribute Name: NumberType
Attribute Value: 10
Attribute DataType: Number
Body: Message body regarding sqs message
Deleted message with handler AQEBXSZRwAN5a4h8D0oXpoQ9WItjbd7ZZGG66ZeKcIWjOMUxkG28Z6K9iG6WFa6giAM5w3Yt1v/qLhvO38gZNuwm6aV8g5CxImAEGNkmHWz2sgv/aiDiTUIahD5JbrfAIVPDFVGNMFth9FBL80wQDeWmxLg8Ha3+QMkB6wh8XC7Bps5AwkeLXjg2o4w/u4TlSxXdCkQttbTW2dKiCujTxzvQ5q/SYi5pnH95UD/3ef4n1W3KOCmTswQVZKrdRDyougdXD+9ZLSd1D85zUPTO1PGoFcvgmQf8n7KSyoRepl/tjb6V5dI3u00ckclk90V+dFu0pMZBbg3/N9AfReOGBmJGcl6VsCleGbKeTI/8X84imWj53AlYc1DRM0SpBOGVr
```


This is interesting, but now I wanted to move this to work with a lambda function, to process the data

Create a lambda function with 
- x86 instruction set
- Author from scratch set to true
- Runtime set to go 1.X


Set the handler inside runtime settings to build/main 
Then proceed to create main.go inside cmd/main.go
main.go:
```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

type Response struct {
	Name       string `json:"author"`
	Lambda     string `json:"title"`
	NumberType string `json:"numberType"`
	TimeStamp  string `json:"timeStamp"`
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	var records []Response
	for _, message := range sqsEvent.Records {
		fmt.Printf("The message %s for event source %s = %s \n", message.MessageId, message.EventSource, message.Body)
		records = append(records, Response{
			Name:       *message.MessageAttributes["Name"].StringValue,
			NumberType: *message.MessageAttributes["NumberType"].StringValue,
			Lambda:     *message.MessageAttributes["Lambda"].StringValue,
			TimeStamp:  message.Attributes["TimeStamp"],
		})
	}
	body, err := json.Marshal(records)
	if err != nil {
		return err
	}
	log.Println(string(body))
	return nil
}
func main() {
	lambda.Start(handler)
}

```

I also setup this build script for easy builds.

Notice building for the x86 architecture and disabling CGO, as this appeared to not work for me

```sh
GOOS=linux CGO_ENABLED=0 go build -o build/main cmd/main.go
echo "Built"
zip build/main.zip build/main
```

(Inside the lambda function, go to configuration and permissions, then create a role inside IAM manager that allows ```AmazonSQSFullAccess```  and add it to the lambda function)

Then add the sqs as a trigger for your function 

Upload your file to the console ( the zip created in build/)


Also here I had an issue where the role I created couldn't write to CloudWatch logs
After editing the role to have  the correct permissions it worked

```
Your function doesn't have permission to write to Amazon CloudWatch Logs. To view logs, add the **AWSLambdaBasicExecutionRole** managed policy to its execution role. [Open the IAM console]
```


Now if we run our send code again, we can see that inside CloudWatch logs we have the following output:
```
2023/09/12 14:22:32 [{
    "author": "Bob",
    "title": "Physics symbol",
    "numberType": "10",
    "timeStamp": ""
}]

```


Congratulations, you just wrote code to send an SQS message and receive it in a lambda function 