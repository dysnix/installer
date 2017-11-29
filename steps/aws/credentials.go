package aws

import (
	"errors"
	"strings"
	"encoding/base64"

	"golang.org/x/crypto/ssh"
	"git.arilot.com/kuberstack/kuberstack-installer/db"
	"git.arilot.com/kuberstack/kuberstack-installer/savedstate"
	"git.arilot.com/kuberstack/kuberstack-installer/steps"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/powerman/structlog"
)

// SaveCredentials save AWS credentials to the DB for the future use
func SaveCredentials(
	conn db.Connect,
	accessKey string,
	secretKey string,
	region string,
	sshKey string,
	principal savedstate.Principal,
) error {
	sess, err := steps.AwsSession(accessKey, secretKey, region)
	if err != nil {
		return err
	}

	_, err = ec2.New(sess).DescribeRegions(nil)
	if err != nil {
		if strings.HasPrefix(err.Error(), "AuthFailure") {
			structlog.DefaultLogger.PrintErr(err)
			return errors.New("AWS credentials are not valid")
		}
		return err
	}

	err = validatePubKey(sshKey)
	if err != nil {
		structlog.DefaultLogger.PrintErr("Error validate SSH public key")
		return errors.New("SSH public key are not valid")
	}

	principal.Sess.AccessKey = accessKey
	principal.Sess.SecretKey = secretKey
	principal.Sess.Region = region
	principal.Sess.SSHPubKey = sshKey

	return conn.SaveState(principal.ID, principal.Sess)
}

func validatePubKey(sshPubKey string) (err error) {
	parts := strings.Fields(sshPubKey)

	if len(parts) < 2 {
		return errors.New("SSH public key are not valid")
	}

	key, _ := base64.StdEncoding.DecodeString(parts[1])
	_, err = ssh.ParsePublicKey(key)

	if err != nil {
		return errors.New("SSH public key are not valid")
	}

	return nil
}
