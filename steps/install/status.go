package install

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/powerman/structlog"

	"git.arilot.com/kuberstack/kuberstack-installer/savedstate"
	"net"
)

type statusType int8

// Possible statuses for the ongoing install
const (
	StatusInitial statusType = 0
	StatusCreated statusType = 1
	StatusUpdated statusType = 2
	StatusRolled  statusType = 3
	StatusReady   statusType = 4
	StatusFailed  statusType = -1
)

var statuses struct {
	statuses map[string]statusType
	sync.RWMutex
}

func init() {
	statuses.statuses = make(map[string]statusType, 16)
}

func setStatus(id string, status statusType) {
	statuses.Lock()
	defer statuses.Unlock()

	statuses.statuses[id] = status
}

func getStatus(id string) statusType {
	statuses.Lock()
	defer statuses.Unlock()

	return statuses.statuses[id]
}

var readyRegexp = regexp.MustCompile(`NODE\s+STATUS\s*\nNAME\s+ROLE\s+READY\s*\n(?:[^\s]+\s+(?:(?:node)|(?:master))\s+True\s*\n)+\s*\n`)

// GetStatus returns a status of the ongoing install
func GetStatus(
	principal savedstate.Principal,
	itself string,
	tmpDir string,
	timeout time.Duration,
	logger *structlog.Logger,
) (statusType, statusType) {
	status := getStatus(principal.ID)
	if status < StatusUpdated || status == StatusReady {
		return StatusReady, status
	}

	clusterName := principal.Sess.Name + "." + principal.Sess.Domain
	if strings.HasSuffix(clusterName, ".") {
		clusterName = clusterName[:len(clusterName)-1]
	}
	apiHost := "api." + clusterName

	res, err := net.LookupHost(apiHost)
	if err != nil {
		logger.Debug("Kubernetes API host has not domain resolve", "Host", apiHost)
		return StatusReady, status
	}
	logger.Debug("Kubernetes API host resolved to", res)

	homeDir := filepath.Join(tmpDir, principal.ID)

	// Validate //////////////////////////////////////////////////////////////
	cmdParams := []string{
		"--kopsValidate",
		fmt.Sprintf("--name=%v", clusterName),
		fmt.Sprintf("--state=s3://%v", principal.Sess.Bucket),
		fmt.Sprintf("--timeout=%v", timeout),
	}

	logger.Debug("Calling Kops validate", "params", cmdParams)

	cmd := exec.Command(itself, cmdParams...) // #nosec
	cmd.Env = []string{
		fmt.Sprintf("HOME=%v", homeDir),
		// ToDo: replace with file in $HOME
		fmt.Sprintf("AWS_ACCESS_KEY=%v", principal.Sess.AccessKey),
		fmt.Sprintf("AWS_SECRET_KEY=%v", principal.Sess.SecretKey),
	}

	cmdOut, err := cmd.CombinedOutput()

	if err != nil {
		logger.PrintErr("Kops validate failed", "err", err, "out", string(cmdOut))
		return StatusReady, status
	}

	logger.Debug("Kops validate done", "out", string(cmdOut))

	if !readyRegexp.Match(cmdOut) {
		logger.Debug("Cluster is not ready yet", "total", StatusReady, "current", status)
		return StatusReady, status
	}

	setStatus(principal.ID, StatusReady)

	logger.Info("Cluster ready")

	return StatusReady, StatusReady
}

// DropStatus removes a status from the table
func DropStatus(id string) {
	statuses.RLock()
	defer statuses.RUnlock()

	delete(statuses.statuses, id)
}
