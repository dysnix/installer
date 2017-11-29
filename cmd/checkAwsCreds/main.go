package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	kuberstackAWS "git.arilot.com/kuberstack/kuberstack-installer/steps/aws"
)

func main() {
	ParseFlags()

	sess, err := session.NewSession(
		&aws.Config{
			Region: aws.String(kuberstackAWS.GetRegions()[0]),
			Credentials: credentials.NewStaticCredentials(
				*Flags.AccessKey,
				*Flags.SecretKey,
				*Flags.Token,
			),
		},
	)
	if err != nil {
		panic(err)
	}

	// Create new EC2 client
	svc := ec2.New(sess)

	resultRegions, err := svc.DescribeRegions(nil)
	if err != nil {
		panic(err)
	}

	for _, region := range resultRegions.Regions {
		fmt.Printf("Region %q\n", *region.RegionName)
	}
}
