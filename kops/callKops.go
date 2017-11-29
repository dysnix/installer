package kops

import (
	"fmt"
	"os"
	"time"

	"git.arilot.com/kuberstack/kopsEmbeded"
	"github.com/powerman/structlog"
)

// CheckKopsRun checks if the app called for kops functions
// and run a corresponding one if so
func CheckKopsRun(
	kopsConfig Config,
	cmdItself string,
	logger *structlog.Logger,
) (int, bool) {
	switch {
	case kopsConfig.KopsCreate:
		logger.Debug("Kops called", "cmd", cmdItself, "params", kopsConfig)
		err := ExecuteCreate(kopsConfig, logger)
		if err != nil {
			logger.PrintErr("Kops create", "err", err)
			return 1, true
		}
		return 0, true
	case kopsConfig.KopsUpdate:
		logger.Debug("Kops called", "cmd", cmdItself, "params", kopsConfig)
		err := ExecuteUpdate(kopsConfig, logger)
		if err != nil {
			logger.PrintErr("Kops update", "err", err)
			return 2, true
		}
		return 0, true
	case kopsConfig.KopsRolling:
		logger.Debug("Kops called", "cmd", cmdItself, "params", kopsConfig)
		err := ExecuteRolling(kopsConfig, logger)
		if err != nil {
			logger.PrintErr("Kops rolling", "err", err)
			return 3, true
		}
		return 0, true
	case kopsConfig.KopsValidate:
		logger.Debug("Kops called", "cmd", cmdItself, "params", kopsConfig)
		err := ExecuteValidate(kopsConfig, logger)
		if err != nil {
			logger.PrintErr("Kops validate", "err", err)
			return 4, true
		}
		return 0, true
	case kopsConfig.KopsDelete:
		logger.Debug("Kops called", "cmd", cmdItself, "params", kopsConfig)
		err := ExecuteDelete(kopsConfig, logger)
		if err != nil {
			logger.PrintErr("Kops delete", "err", err)
			return 5, true
		}
		return 0, true
		// case kubectlConfig.KubectlGetNodes:
		// 	logger.Debug("Kubectl called", "cmd", cmdItself)
		// 	err := kubectl.GetNodes(kubectlConfig, logger)
		// 	if err != nil {
		// 		logger.Fatalf("Error executing embeded kops update: %v", err)
		// 	}
		// 	os.Exit(0)
	}

	return -1, false
}

// Config is a command-line config for embeded kops
type Config struct {
	KopsCreate   bool          `long:"kopsCreate" description:"run embedded kops binary to create a cluster"`
	KopsUpdate   bool          `long:"kopsUpdate" description:"run embedded kops binary to run/update a cluster"`
	KopsRolling  bool          `long:"kopsRolling" description:"run embedded kops binary to rolling update a cluster"`
	KopsValidate bool          `long:"kopsValidate" description:"run embedded kops binary to validate a cluster"`
	KopsDelete   bool          `long:"kopsDelete" description:"run embedded kops binary to delete a cluster"`
	TmpDir       string        `long:"tmpDir" description:"directory to save the SSH pub keys to be passed to kops" default:"./"`
	Timeout      time.Duration `long:"timeout" description:"Max time kops command allowed to execute" default:"120s"`

	Zones              string `long:"zones" description:"Zones in which to run the cluster"`
	Name               string `long:"name" description:"Name of cluster"`
	State              string `long:"state" description:"Location of state storage"`
	MasterCount        int32  `long:"master-count" description:"Set the number of masters"`
	MasterSize         string `long:"master-size" description:"Set instance size for masters"`
	MasterVolumeSize   int32  `long:"master-volume-size" description:"Set instance volume size (in GB) for masters"`
	MasterZones        string `long:"master-zones" description:"Zones in which to run masters (must be an odd number)"`
	NodeCount          int32  `long:"node-count" description:"Set the number of nodes"`
	NodeSecurityGroups string `long:"node-security-groups" description:"Add precreated additional security groups to nodes."`
	NodeSize           string `long:"node-size" description:"Set instance size for nodes"`
	NodeVolumeSize     int32  `long:"node-volume-size" description:"Set instance volume size (in GB) for nodes	"`
	SSHPublicKey       string `long:"ssh-public-key" description:"SSH public key to use"`
}

// ExecuteCreate calls an embeded kops create cluster with the params provided
func ExecuteCreate(kopsConfig Config, logger *structlog.Logger) error {
	time.AfterFunc(
		kopsConfig.Timeout,
		func() {
			_, err := fmt.Fprintf(os.Stderr, "Timeout (%v) exceeded\n", kopsConfig.Timeout)
			if err != nil {
				panic(err)
			}
			os.Exit(9)
		},
	)

	params := []string{
		"create",
		"cluster",
		"--admin-access=0.0.0.0/0",
		"--api-loadbalancer-type=public",
		"--associate-public-ip=true",
		"--authorization=AlwaysAllow",
		"--channel=stable",
		"--cloud=aws",
		"--dns=public",
		"--model=config,proto,cloudup",
		"--ssh-access=0.0.0.0/0",
		"--target=direct",
		"--topology=public",
		"--yes",
		"--networking=kubenet",
		fmt.Sprintf("--zones=%v", kopsConfig.Zones),
		fmt.Sprintf("--name=%v", kopsConfig.Name),
		fmt.Sprintf("--state=%v", kopsConfig.State),
		fmt.Sprintf("--master-count=%v", kopsConfig.MasterCount),
		fmt.Sprintf("--master-size=%v", kopsConfig.MasterSize),
		fmt.Sprintf("--master-volume-size=%v", kopsConfig.MasterVolumeSize),
		fmt.Sprintf("--master-zones=%v", kopsConfig.MasterZones),
		fmt.Sprintf("--node-count=%v", kopsConfig.NodeCount),
		fmt.Sprintf("--node-security-groups=%v", kopsConfig.NodeSecurityGroups),
		fmt.Sprintf("--node-size=%v", kopsConfig.NodeSize),
		fmt.Sprintf("--node-volume-size=%v", kopsConfig.NodeVolumeSize),
		fmt.Sprintf("--ssh-public-key=%v", kopsConfig.SSHPublicKey),
		"--logtostderr",
	}

	logger.Debug("Calling embeded kops", "params", params)

	return kopsEmbeded.Execute(params...)
}

// ExecuteUpdate calls an embeded kops update cluster
func ExecuteUpdate(kopsConfig Config, logger *structlog.Logger) error {
	time.AfterFunc(
		kopsConfig.Timeout,
		func() {
			_, err := fmt.Fprintf(os.Stderr, "Timeout (%v) exceeded\n", kopsConfig.Timeout)
			if err != nil {
				panic(err)
			}
			os.Exit(9)
		},
	)

	params := []string{
		"update",
		"cluster",
		"--yes",
		fmt.Sprintf("--name=%v", kopsConfig.Name),
		fmt.Sprintf("--state=%v", kopsConfig.State),
	}

	logger.Debug("Calling embeded kops", "params", params)

	return kopsEmbeded.Execute(params...)
}

// ExecuteRolling calls an embeded kops rolling update cluster
func ExecuteRolling(kopsConfig Config, logger *structlog.Logger) error {
	time.AfterFunc(
		kopsConfig.Timeout,
		func() {
			_, err := fmt.Fprintf(os.Stderr, "Timeout (%v) exceeded\n", kopsConfig.Timeout)
			if err != nil {
				panic(err)
			}
			os.Exit(9)
		},
	)

	params := []string{
		"rolling-update",
		"cluster",
		"--yes",
		"--force",
		"--master-interval=5m0s",
		"--node-interval=2m0s",
		fmt.Sprintf("--name=%v", kopsConfig.Name),
		fmt.Sprintf("--state=%v", kopsConfig.State),
	}

	logger.Debug("Calling embeded kops", "params", params)

	return kopsEmbeded.Execute(params...)
}

// ExecuteValidate calls an embeded kops validate cluster with the params provided
func ExecuteValidate(kopsConfig Config, logger *structlog.Logger) error {
	time.AfterFunc(
		kopsConfig.Timeout,
		func() {
			_, err := fmt.Fprintf(os.Stderr, "Timeout (%v) exceeded\n", kopsConfig.Timeout)
			if err != nil {
				panic(err)
			}
			os.Exit(9)
		},
	)

	params := []string{
		"validate",
		"cluster",
		fmt.Sprintf("--name=%v", kopsConfig.Name),
		fmt.Sprintf("--state=%v", kopsConfig.State),
	}

	logger.Debug("Calling embeded kops", "params", params)

	return kopsEmbeded.Execute(params...)
}

// ExecuteDelete calls an embeded kops delete cluster with the params provided
func ExecuteDelete(kopsConfig Config, logger *structlog.Logger) error {
	time.AfterFunc(
		kopsConfig.Timeout,
		func() {
			_, err := fmt.Fprintf(os.Stderr, "Timeout (%v) exceeded\n", kopsConfig.Timeout)
			if err != nil {
				panic(err)
			}
			os.Exit(9)
		},
	)

	params := []string{
		"delete",
		"cluster",
		fmt.Sprintf("--name=%v", kopsConfig.Name),
		fmt.Sprintf("--state=%v", kopsConfig.State),
		"--yes",
	}

	logger.Debug("Calling embeded kops", "params", params)

	return kopsEmbeded.Execute(params...)
}
