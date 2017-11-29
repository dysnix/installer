package install

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/powerman/structlog"

	"git.arilot.com/kuberstack/kuberstack-installer/db"
	"git.arilot.com/kuberstack/kuberstack-installer/savedstate"
	"git.arilot.com/kuberstack/kuberstack-installer/steps/auth"
)

const (
	sshKeyFile string = "id.pub"
)

// Install runs an installation process
func Install(
	conn db.Connect,
	principal savedstate.Principal,
	itself string,
	tmpDir string,
	timeout time.Duration,
	logger *structlog.Logger,
) error {
	logger = logger.New("id", principal.ID).AppendPrefixKeys("id")

	DropStatus(principal.ID)

	homeDir := filepath.Join(tmpDir, principal.ID)

	err := os.RemoveAll(homeDir)
	if err != nil {
		logger.PrintErr("Kops home dir cleanup error", "err", err)
		return fmt.Errorf("Internal server error")
	}

	err = os.MkdirAll(homeDir, 0700)
	if err != nil {
		logger.PrintErr("Kops home dir creation error", "err", err)
		return fmt.Errorf("Internal server error")
	}

	// ToDo: replace by pipe
	err = saveSSHkey(principal.Sess.SSHPubKey, filepath.Join(homeDir, sshKeyFile), logger)
	if err != nil {
		return err
	}
	logger.Debug("SSH key saved", "file", filepath.Join(homeDir, sshKeyFile))

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

	go doInstall(
		conn,
		principal.ID,
		homeDir,
		itself,
		timeout,
		clusterName,
		principal.Sess,
		logger,
	)

	return conn.SaveState(principal.ID, principal.Sess)
}

func saveSSHkey(key string, fileName string, logger *structlog.Logger) error {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		logger.PrintErr("SSH key file create error", "err", err, "file", fileName)
		return fmt.Errorf("Internal server error")
	}
	defer mustClose(file)

	_, err = file.WriteString(key)
	if err != nil {
		logger.PrintErr("SSH key write error", "err", err, "file", fileName)
		return fmt.Errorf("Internal server error")
	}

	return nil
}

func mustClose(f io.Closer) {
	err := f.Close()
	if err != nil {
		panic(err)
	}
}

func doInstall(
	conn db.Connect,
	id string,
	homeDir string,
	itself string,
	timeout time.Duration,
	clusterName string,
	sess *savedstate.State,
	logger *structlog.Logger,
) {
	// Create //////////////////////////////////////////////////////////////
	cmdParams := []string{
		"--kopsCreate",
		fmt.Sprintf("--name=%v", clusterName),
		fmt.Sprintf("--timeout=%v", timeout),
		fmt.Sprintf("--state=s3://%v", sess.Bucket),
		fmt.Sprintf("--master-count=%v", sess.Master.Quantity),
		fmt.Sprintf("--master-size=%v", sess.Master.Type),
		fmt.Sprintf("--master-volume-size=%v", sess.Master.StorageSize),
		fmt.Sprintf("--master-zones=%v", strings.Join(sess.Master.Zones, ",")),
		fmt.Sprintf("--node-count=%v", sess.Nodes.Quantity),
		fmt.Sprintf("--node-size=%v", sess.Nodes.Type),
		fmt.Sprintf("--node-volume-size=%v", sess.Nodes.StorageSize),
		fmt.Sprintf("--zones=%v", strings.Join(sess.Nodes.Zones, ",")),
		fmt.Sprintf("--ssh-public-key=%v", filepath.Join(homeDir, sshKeyFile)),
	}

	cmdEnv := []string{
		fmt.Sprintf("HOME=%v", homeDir),
		// ToDo: replace with file in $HOME
		fmt.Sprintf("AWS_ACCESS_KEY=%v", sess.AccessKey),
		fmt.Sprintf("AWS_SECRET_KEY=%v", sess.SecretKey),
	}

	cmd := exec.Command(itself, cmdParams...) // #nosec
	cmd.Env = cmdEnv

	logger.Debug("Calling Kops create", "params", cmdParams)
	cmdOut, err := cmd.CombinedOutput()
	if err != nil {
		logger.PrintErr("Kops create failed", "err", err, "out", string(cmdOut))
		setStatus(id, StatusFailed)
		return
	}

	logger.Debug("Kops create done", "out", string(cmdOut))

	// Save config //////////////////////////////////////////////////////////////
	kubecfgName := filepath.Join(homeDir, ".kube", "config")
	kubecfg, err := ioutil.ReadFile(kubecfgName)
	if err != nil {
		logger.PrintErr("Reading kubecfg error", "err", err, "file", kubecfgName)
		setStatus(id, StatusFailed)
		return
	}

	principal, err := auth.GetSession(conn, id)
	if err != nil {
		logger.PrintErr("Reading DB record error", "err", err)
		setStatus(id, StatusFailed)
		return
	}

	if principal == nil {
		logger.PrintErr("Empty DB record error")
		setStatus(id, StatusFailed)
		return
	}

	principal.Sess.Kubecfg = kubecfg

	err = conn.SaveState(principal.ID, principal.Sess)
	if err != nil {
		logger.PrintErr("Saving DB record error", "err", err)
		setStatus(id, StatusFailed)
		return
	}

	setStatus(id, StatusCreated)

	logger.Debug("Kubecfg saved to db", "len", len(principal.Sess.Kubecfg))

	// Update //////////////////////////////////////////////////////////////
	cmdParams = []string{
		"--kopsUpdate",
		fmt.Sprintf("--name=%v", clusterName),
		fmt.Sprintf("--state=s3://%v", sess.Bucket),
		fmt.Sprintf("--timeout=%v", timeout),
	}

	logger.Debug("Calling Kops update", "params", cmdParams)
	cmd = exec.Command(itself, cmdParams...) // #nosec
	cmd.Env = cmdEnv

	cmdOut, err = cmd.CombinedOutput()
	if err != nil {
		logger.PrintErr("Kops update failed", "err", err, "out", string(cmdOut))
		setStatus(id, StatusFailed)
		return
	}

	logger.Debug("Kops update done", "out", string(cmdOut))

	// // Rolling //////////////////////////////////////////////////////////////
	// cmdParams = []string{
	// 	"--kopsRolling",
	// 	fmt.Sprintf("--name=%v", clusterName),
	// 	fmt.Sprintf("--state=s3://%v", sess.Bucket),
	// 	fmt.Sprintf("--timeout=%v", timeout),
	// }
	//
	// 	logger.Debug("Calling Kops rolling", "params", cmdParams)
	// 	cmd = exec.Command(itself, cmdParams...) // #nosec
	// 	cmd.Env = cmdEnv
	//
	// 	cmdOut, err = cmd.CombinedOutput()
	// 	if err != nil {
	// 		logger.PrintErr("Kops rolling failed", "err", err, "out", string(cmdOut))
	// 		setStatus(id, StatusFailed)
	// 		return
	// 	}
	//
	// 	logger.Debug("Kops update done", "out", string(cmdOut))

	setStatus(id, StatusRolled)
}
