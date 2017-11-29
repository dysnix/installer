package install

import "git.arilot.com/kuberstack/kuberstack-installer/savedstate"

// GetKubecfg returns a saved kube config
func GetKubecfg(principal savedstate.Principal) []byte {
	return principal.Sess.Kubecfg
}
