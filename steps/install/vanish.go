package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/powerman/structlog"

	awsSdk "github.com/aws/aws-sdk-go/aws"
	"git.arilot.com/kuberstack/kuberstack-installer/db"
	"git.arilot.com/kuberstack/kuberstack-installer/savedstate"
	"github.com/aws/aws-sdk-go/service/route53"

	"git.arilot.com/kuberstack/kuberstack-installer/steps"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/aws/client"
	"crypto/sha256"
	"encoding/hex"
)

var (
	num1str = "1"
)

// Vanish runs an cluster delete
func Vanish(
	conn db.Connect,
	principal savedstate.Principal,
	itself string,
	tmpDir string,
	timeout time.Duration,
	logger *structlog.Logger,
) error {
	logger = logger.New("id", principal.ID).AppendPrefixKeys("id")

	homeDir := filepath.Join(tmpDir, principal.ID)

	err := os.RemoveAll(homeDir)
	if err != nil {
		logger.PrintErr("Kops home dir cleanup error", "err", err)
		return fmt.Errorf("Internal server error")
	}

	notSetErr := make([]string, 0, 11)

	if len(principal.Sess.Name) == 0 {
		notSetErr = append(notSetErr, "Name")
	}
	if len(principal.Sess.Domain) == 0 {
		notSetErr = append(notSetErr, "Domain")
	}
	if len(principal.Sess.Bucket) == 0 {
		notSetErr = append(notSetErr, "Bucket")
	}
	if principal.Sess.Master.Quantity == 0 {
		notSetErr = append(notSetErr, "Master.Quantity")
	}
	if len(principal.Sess.Master.Type) == 0 {
		notSetErr = append(notSetErr, "Master.Type")
	}
	if principal.Sess.Master.StorageSize == 0 {
		notSetErr = append(notSetErr, "Master.StorageSize")
	}
	if len(principal.Sess.Master.Zones) == 0 {
		notSetErr = append(notSetErr, "Master.Zones")
	}
	if principal.Sess.Nodes.Quantity == 0 {
		notSetErr = append(notSetErr, "Nodes.Quantity")
	}
	if len(principal.Sess.Nodes.Type) == 0 {
		notSetErr = append(notSetErr, "Nodes.Type")
	}
	if principal.Sess.Nodes.StorageSize == 0 {
		notSetErr = append(notSetErr, "Nodes.StorageSize")
	}
	if len(principal.Sess.Nodes.Zones) == 0 {
		notSetErr = append(notSetErr, "Nodes.Zones")
	}
	if len(principal.Sess.SSHPubKey) == 0 {
		notSetErr = append(notSetErr, "SSH public key")
	}

	if len(notSetErr) > 0 {
		return logger.Err(fmt.Errorf("Requred parameter(s) not set: %v", notSetErr))
	}

	clusterName := principal.Sess.Name + "." + principal.Sess.Domain
	if strings.HasSuffix(clusterName, ".") {
		clusterName = clusterName[:len(clusterName)-1]
	}

	go doDelete(
		conn,
		principal.ID,
		homeDir,
		itself,
		timeout,
		clusterName,
		principal.Sess,
		logger,
	)

	DropStatus(principal.ID)

	return nil
}

func doDelete(
	conn db.Connect,
	id string,
	homeDir string,
	itself string,
	timeout time.Duration,
	clusterName string,
	sess *savedstate.State,
	logger *structlog.Logger,
) error {
	cmdParams := []string{
		"--kopsDelete",
		fmt.Sprintf("--name=%v", clusterName),
		fmt.Sprintf("--state=s3://%v", sess.Bucket),
	}

	cmdEnv := []string{
		fmt.Sprintf("HOME=%v", homeDir),
		// ToDo: replace with file in $HOME
		fmt.Sprintf("AWS_ACCESS_KEY=%v", sess.AccessKey),
		fmt.Sprintf("AWS_SECRET_KEY=%v", sess.SecretKey),
	}

	cmd := exec.Command(itself, cmdParams...) // #nosec
	cmd.Env = cmdEnv

	logger.Debug("Calling Kops delete", "params", cmdParams)
	cmdOut, err := cmd.CombinedOutput()
	if err != nil {
		logger.PrintErr("Kops delete failed", "err", err, "out", string(cmdOut))
		return err
	}

	logger.Debug("Kops delete done", "out", string(cmdOut))

	// Remove dns zone
	awsSess, err := steps.AwsSession(
		sess.AccessKey,
		sess.SecretKey,
		sess.Region,
	)
	if err != nil {
		logger.PrintErr(err)
		return err
	}

	r53 := route53.New(awsSess)

	domainName := sess.Name + "." + sess.Domain + "."
	zoneID, DNSZoneDeleteStatus, err := deleteZone(r53, domainName)
	if err != nil {
		logger.PrintErr(err)
		return err
	}

	logger.Debug("Zone deleted", "name", domainName, "Id", zoneID, "status", DNSZoneDeleteStatus, "cluster", clusterName)

	err = deleteBucket(logger, awsSess, id, domainName)
	if err != nil {
		logger.PrintErr(err)
		return err
	}

	logger.Debug("S3 bucket deleted", "cluster", clusterName)

	return nil
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

func deleteZone(r53 *route53.Route53, name string) (string, string, error) {
	zoneID, err := getZoneID(r53, name)
	if err != nil {
		return "", "", err
	}
	if zoneID == "" {
		return "", "", fmt.Errorf("Zone does not exist: %q", name)
	}

	res, err := r53.DeleteHostedZone(
		&route53.DeleteHostedZoneInput{
			Id: &zoneID,
		},
	)
	if err != nil {
		return "", "", err
	}

	return zoneID, awsSdk.StringValue(res.ChangeInfo.Status), nil
}

func deleteBucket(logger *structlog.Logger, sess client.ConfigProvider, id string, domain string) (error) {
	clnS3 := s3.New(sess)

	idHash := sha256.Sum224([]byte(id))

	bucketName := hex.EncodeToString(idHash[:])

	// Empty bucket
	logger.Debug("removing objects from S3 bucket :", bucketName)

	params := &s3.ListObjectsInput{
		Bucket: awsSdk.String(bucketName),
	}
	for {
		objects, err := clnS3.ListObjects(params)
		if err != nil {
			return err
		}
		//Checks if the bucket is already empty
		if len((*objects).Contents) == 0 {
			logger.Debug("Bucket is already empty", "bucket", bucketName)
			break
		}
		logger.Debug("First object in batch | ", *(objects.Contents[0].Key))

		//creating an array of pointers of ObjectIdentifier
		objectsToDelete := make([]*s3.ObjectIdentifier, 0, 1000)
		for _, object := range (*objects).Contents {
			obj := s3.ObjectIdentifier{
				Key: object.Key,
			}
			objectsToDelete = append(objectsToDelete, &obj)
		}
		//Creating JSON payload for bulk delete
		deleteArray := s3.Delete{Objects: objectsToDelete}
		deleteParams := &s3.DeleteObjectsInput{
			Bucket: awsSdk.String(bucketName),
			Delete: &deleteArray,
		}
		//Running the Bulk delete job (limit 1000)
		_, err = clnS3.DeleteObjects(deleteParams)
		if err != nil {
			return err
		}
		if *(*objects).IsTruncated { //if there are more objects in the bucket, IsTruncated = true
			params.Marker = (*deleteParams).Delete.Objects[len((*deleteParams).Delete.Objects)-1].Key
			logger.Debug("Requesting next batch | ", *(params.Marker))
		} else { //if all objects in the bucket have been cleaned up.
			break
		}
	}
	logger.Debug("Emptied S3 bucket", "bucket", bucketName)

	// Remove versions of files
	logger.Debug("removing versions of files from S3 bucket :", bucketName)

	params_versions := &s3.ListObjectVersionsInput{
		Bucket: awsSdk.String(bucketName),
	}
	for {
		objects_versions, err := clnS3.ListObjectVersions(params_versions)
		if err != nil {
			return err
		}
		//Checks if the bucket is already empty
		if len((*objects_versions).Versions) == 0 && len((*objects_versions).DeleteMarkers) == 0 {
			logger.Debug("Bucket is already empty", "bucket", bucketName)
			break
		}

		//creating an array of pointers of ObjectIdentifier
		objectsToDelete := make([]*s3.ObjectIdentifier, 0, 1000)
		for _, object := range (*objects_versions).Versions {
			obj := s3.ObjectIdentifier{
				Key:       object.Key,
				VersionId: object.VersionId,
			}
			objectsToDelete = append(objectsToDelete, &obj)
		}
		for _, object := range (*objects_versions).DeleteMarkers {
			obj := s3.ObjectIdentifier{
				Key:       object.Key,
				VersionId: object.VersionId,
			}
			objectsToDelete = append(objectsToDelete, &obj)
		}

		//Creating JSON payload for bulk delete
		deleteArray := s3.Delete{Objects: objectsToDelete}
		deleteParams := &s3.DeleteObjectsInput{
			Bucket: awsSdk.String(bucketName),
			Delete: &deleteArray,
		}
		//Running the Bulk delete job (limit 1000)
		_, err = clnS3.DeleteObjects(deleteParams)
		if err != nil {
			return err
		}
		if *(*objects_versions).IsTruncated { //if there are more objects in the bucket, IsTruncated = true
			params.Marker = (*deleteParams).Delete.Objects[len((*deleteParams).Delete.Objects)-1].Key
			logger.Debug("Requesting next batch | ", *(params.Marker))
		} else { //if all objects in the bucket have been cleaned up.
			break
		}
	}
	logger.Debug("Emptied S3 versions of files from bucket", "bucket", bucketName)

	// Delete bucket
	_, err := clnS3.DeleteBucket(
		&s3.DeleteBucketInput{
			Bucket: awsSdk.String(bucketName),
		},
	)

	if err != nil {
		return err
	}

	return err
}