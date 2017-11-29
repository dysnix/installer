package nodes

import (
	"git.arilot.com/kuberstack/kuberstack-installer/savedstate"
	"git.arilot.com/kuberstack/kuberstack-installer/steps"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// GetZones returns a list of availability zones for the particular region
func GetZones(
	region string,
	principal savedstate.Principal,
) ([]string, error) {
	sess, err := steps.AwsSession(principal.Sess.AccessKey, principal.Sess.SecretKey, region)
	if err != nil {
		return nil, err
	}

	zones, err := ec2.New(sess).DescribeAvailabilityZones(nil)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(zones.AvailabilityZones))

	for _, zone := range zones.AvailabilityZones {
		result = append(result, *zone.ZoneName)
	}

	return result, nil
}
