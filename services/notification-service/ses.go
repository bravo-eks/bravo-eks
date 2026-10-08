package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sesv2types "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type SESClient struct {
	client  *sesv2.Client
	from    string
	replyTo string
}

func newSESClient(ctx context.Context) (*SESClient, error) {
	region := getEnv("AWS_REGION", "eu-west-2")
	from := getEnv("SES_FROM_EMAIL", "notifications@bravoorders.com")
	replyTo := getEnv("SES_REPLY_TO_EMAIL", "support@bravoorders.com")

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	return &SESClient{
		client:  sesv2.NewFromConfig(cfg),
		from:    from,
		replyTo: replyTo,
	}, nil
}

func (s *SESClient) SendEmail(
	ctx context.Context,
	recipient string,
	subject string,
	body string,
) (string, error) {
	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.from),
		Destination: &sesv2types.Destination{
			ToAddresses: []string{recipient},
		},
		ReplyToAddresses: []string{s.replyTo},
		Content: &sesv2types.EmailContent{
			Simple: &sesv2types.Message{
				Subject: &sesv2types.Content{
					Data: aws.String(subject),
				},
				Body: &sesv2types.Body{
					Text: &sesv2types.Content{
						Data: aws.String(body),
					},
				},
			},
		},
	}

	output, err := s.client.SendEmail(ctx, input)
	if err != nil {
		return "", fmt.Errorf("send email through SES: %w", err)
	}

	if output.MessageId == nil || *output.MessageId == "" {
		return "", fmt.Errorf("SES returned an empty message ID")
	}

	return *output.MessageId, nil
}
