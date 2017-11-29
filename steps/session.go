package steps

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

// AwsSession creates an AWS session for the given credentials
func AwsSession(
	id string,
	secret string,
	region string,
) (*session.Session, error) {
	return session.NewSession(
		&aws.Config{
			Region: aws.String(region),
			Credentials: credentials.NewStaticCredentials(
				id,
				secret,
				"",
			),
		},
	)
}
