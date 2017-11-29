package aws

import "github.com/aws/aws-sdk-go/aws/endpoints"

// GetRegions returns all the regions for the EC2 service in AWS partition
func GetRegions() []string {
	regions := endpoints.AwsPartition().Services()[endpoints.Ec2ServiceID].Regions()

	ids := make([]string, 0, len(regions))
	for id := range regions {
		ids = append(ids, id)
	}

	return ids
}
