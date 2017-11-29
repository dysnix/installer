package cluster

import (
	"git.arilot.com/kuberstack/kuberstack-installer/db"
	"git.arilot.com/kuberstack/kuberstack-installer/savedstate"
	"git.arilot.com/kuberstack/kuberstack-installer/steps/install"
)

// Save saves the cluster params for the future use
func Save(
	conn db.Connect,
	domain string,
	name string,
	clustType int64,
	principal savedstate.Principal,
) error {
	principal.Sess.Domain = domain
	principal.Sess.Name = name
	principal.Sess.Type = clustType

	install.DropStatus(principal.ID)

	return conn.SaveState(principal.ID, principal.Sess)
}
