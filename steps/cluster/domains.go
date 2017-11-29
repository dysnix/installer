package cluster

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/powerman/structlog"
	"github.com/satori/go.uuid"

	"git.arilot.com/kuberstack/kuberstack-installer/db"
	"git.arilot.com/kuberstack/kuberstack-installer/savedstate"
	"git.arilot.com/kuberstack/kuberstack-installer/steps"

	awsSdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	num1str        = "1"
	listDomainsMax = "10000"
	int300         = int64(300)
	strCREATE      = "CREATE"
	strNS          = "NS"
)

// CheckDomain check the new domain name uniqueness
// and creates a new hosted zone in case it dows not exists yet.
func CheckDomain(
	conn db.Connect,
	domain string,
	name string,
	principal savedstate.Principal,
) error {
	domain, name, newName := fixNames(domain, name)

	sess, err := steps.AwsSession(
		principal.Sess.AccessKey,
		principal.Sess.SecretKey,
		principal.Sess.Region,
	)
	if err != nil {
		return err
	}

	r53 := route53.New(sess)

	domZoneID, err := getZoneID(r53, domain)
	if err != nil {
		return err
	}
	if domZoneID == "" {
		return fmt.Errorf("Domain is out of control: %q", domain)
	}

	err = checkRecordAvailability(r53, domZoneID, newName)
	if err != nil {
		return err
	}

	zoneID, zoneNSes, zoneWatchID, err := createZone(r53, newName, principal.ID)
	if err != nil {
		return err
	}
	structlog.DefaultLogger.Info("Zone created", "name", newName, "id", zoneID, "ns", zoneNSes, "watch", zoneWatchID)

	recWatchID, err := createNSRecords(r53, domZoneID, newName, zoneNSes, principal.ID)
	if err != nil {
		structlog.DefaultLogger.PrintErr(err)
		return fmt.Errorf("Unexpected error creating NS records for %q", newName)
	}
	structlog.DefaultLogger.Info("NS records created", "name", newName, "ns", zoneNSes, "watch", recWatchID)

	principal.Sess.Name = name
	principal.Sess.Domain = domain
	principal.Sess.ZoneID = zoneID
	principal.Sess.ZoneWatchID = zoneWatchID
	principal.Sess.RecWatchID = recWatchID

	principal.Sess.Bucket, err = createBucket(sess, principal.ID, newName)
	if err != nil {
		structlog.DefaultLogger.PrintErr("Create bucket error", "bucket", principal.Sess.Bucket, "err", err)
		return fmt.Errorf("Internal server error")
	}
	structlog.DefaultLogger.Info("S3 bucket created", "bucket", principal.Sess.Bucket)

	return conn.SaveState(principal.ID, principal.Sess)
}

func getZoneID(r53 *route53.Route53, name string) (string, error) {
	zones, err := r53.ListHostedZonesByName(
		&route53.ListHostedZonesByNameInput{
			DNSName:  &name,
			MaxItems: &num1str,
		},
	)
	if err != nil {
		return "", err
	}

	if len(zones.HostedZones) == 0 || *zones.HostedZones[0].Name != name {
		return "", nil
	}

	return *zones.HostedZones[0].Id, nil
}

func createZone(r53 *route53.Route53, name string, principal string) (string, []string, string, error) {
	zoneID, err := getZoneID(r53, name)
	if err != nil {
		return "", nil, "", err
	}
	if zoneID != "" {
		return "", nil, "", fmt.Errorf("Zone already exists: %q", name)
	}

	res, err := r53.CreateHostedZone(
		&route53.CreateHostedZoneInput{
			Name:            &name,
			CallerReference: awsSdk.String(uuid.NewV4().String()),
			HostedZoneConfig: &route53.HostedZoneConfig{
				Comment: awsSdk.String(
					fmt.Sprintf("Created as part of Kuberstack installation: %q", principal),
				),
			},
		},
	)
	if err != nil {
		return "", nil, "", err
	}

	return awsSdk.StringValue(res.HostedZone.Id),
		awsSdk.StringValueSlice(res.DelegationSet.NameServers),
		awsSdk.StringValue(res.ChangeInfo.Id),
		nil
}

func createNSRecords(
	r53 *route53.Route53,
	zoneID string,
	name string,
	servers []string,
	principal string,
) (string, error) {
	recCreated, err := r53.ChangeResourceRecordSets(
		&route53.ChangeResourceRecordSetsInput{
			HostedZoneId: &zoneID,
			ChangeBatch: &route53.ChangeBatch{
				Comment: awsSdk.String(
					fmt.Sprintf("Created as part of Kuberstack installation: %q", principal),
				),
				Changes: []*route53.Change{
					&route53.Change{
						Action: &strCREATE,
						ResourceRecordSet: &route53.ResourceRecordSet{
							Name:            &name,
							Type:            &strNS,
							TTL:             &int300,
							ResourceRecords: prepareNSRecords(servers),
						},
					},
				},
			},
		},
	)
	if err != nil {
		return "", err
	}

	return awsSdk.StringValue(recCreated.ChangeInfo.Id), nil
}

// GetDomains returns a list of domais registered for the corresponding account
func GetDomains(
	principal savedstate.Principal,
) ([]string, error) {
	sess, err := steps.AwsSession(
		principal.Sess.AccessKey,
		principal.Sess.SecretKey,
		principal.Sess.Region,
	)
	if err != nil {
		return nil, err
	}

	r53 := route53.New(sess)

	zones, err := r53.ListHostedZonesByName(
		&route53.ListHostedZonesByNameInput{
			MaxItems: &listDomainsMax,
		},
	)
	if err != nil {
		return nil, err
	}

	domains := make([]string, 0, len(zones.HostedZones))
	for _, zone := range zones.HostedZones {
		name := *zone.Name
		if strings.HasSuffix(name, ".") {
			name = name[:len(name)-len(".")]
		}
		domains = append(domains, name)
	}

	return domains, nil
}

func fixNames(domain string, name string) (string, string, string) {
	if !strings.HasSuffix(domain, ".") {
		domain += "."
	}

	domain = strings.ToLower(domain)
	name = strings.ToLower(name)
	newName := name + "." + domain

	return domain, name, newName
}

func checkRecordAvailability(r53 *route53.Route53, zoneID string, name string) error {
	recs, err := r53.ListResourceRecordSets(
		&route53.ListResourceRecordSetsInput{
			HostedZoneId:    &zoneID,
			StartRecordName: &name,
			MaxItems:        &num1str,
		},
	)
	if err != nil {
		return err
	}
	if len(recs.ResourceRecordSets) > 0 && *recs.ResourceRecordSets[0].Name == name {
		return fmt.Errorf("Record already exists: %q", name)
	}

	return nil
}

func prepareNSRecords(servers []string) []*route53.ResourceRecord {
	nsRecs := make([]*route53.ResourceRecord, 0, len(servers))

	for _, server := range servers {
		if !strings.HasSuffix(server, ".") {
			server += "."
		}
		nsRecs = append(nsRecs, &route53.ResourceRecord{Value: awsSdk.String(server)})
	}

	return nsRecs
}

// IsDNSInSync checks propagation status for the previously created zone and record
func IsDNSInSync(
	principal savedstate.Principal,
) (bool, error) {
	sess, err := steps.AwsSession(
		principal.Sess.AccessKey,
		principal.Sess.SecretKey,
		principal.Sess.Region,
	)
	if err != nil {
		return false, err
	}

	r53 := route53.New(sess)

	status, err := r53.GetChange(&route53.GetChangeInput{Id: &principal.Sess.ZoneWatchID})
	if err != nil {
		return false, err
	}
	if awsSdk.StringValue(status.ChangeInfo.Status) != "INSYNC" {
		return false, nil
	}

	status, err = r53.GetChange(&route53.GetChangeInput{Id: &principal.Sess.RecWatchID})
	if err != nil {
		return false, err
	}
	if awsSdk.StringValue(status.ChangeInfo.Status) != "INSYNC" {
		return false, nil
	}

	return true, nil
}

func createBucket(sess client.ConfigProvider, id string, domain string) (string, error) {
	clnS3 := s3.New(sess)

	idHash := sha256.Sum224([]byte(id))

	bucketName := hex.EncodeToString(idHash[:])

	_, err := clnS3.CreateBucket(
		&s3.CreateBucketInput{
			Bucket: awsSdk.String(bucketName),
		},
	)
	if err != nil {
		return bucketName, err
	}

	_, err = clnS3.PutBucketVersioning(
		&s3.PutBucketVersioningInput{
			Bucket: awsSdk.String(bucketName),
			VersioningConfiguration: &s3.VersioningConfiguration{
				Status: awsSdk.String("Enabled"),
			},
		},
	)

	return bucketName, err
}
