package nodes

import (
	"fmt"

	"git.arilot.com/kuberstack/kuberstack-installer/db"
	"git.arilot.com/kuberstack/kuberstack-installer/protocol/gen/models"
	"git.arilot.com/kuberstack/kuberstack-installer/savedstate"
)

// MinVolumeSize is a minimum volume size for any instance
const MinVolumeSize = 20

// Save saves cluster config to the DB
func Save(
	conn db.Connect,
	master models.NodesRequest,
	nodes models.NodesRequest,
	principal savedstate.Principal,
) error {
	if !checkMachineType(master.InstanceType) {
		return fmt.Errorf("Instance type not handled: %q", master.InstanceType)
	}
	if !checkMachineType(nodes.InstanceType) {
		return fmt.Errorf("Instance type not handled: %q", nodes.InstanceType)
	}
	if master.StorageSize < MinVolumeSize || nodes.StorageSize < MinVolumeSize {
		return fmt.Errorf("Minimum volume size is %d", MinVolumeSize)
	}
	if master.Instances > 0 &&
		len(master.Zones) > 0 &&
		master.Instances != int64(len(master.Zones)) {
		return fmt.Errorf(
			"specified %d master zones, but also requested %d masters. If specifying both, the count should match",
			len(master.Zones),
			master.Instances,
		)
	}

	principal.Sess.Master.Type = master.InstanceType
	principal.Sess.Master.Quantity = master.Instances
	principal.Sess.Master.Zones = master.Zones
	principal.Sess.Master.StorageSize = master.StorageSize
	principal.Sess.Master.StorageType = master.StorageType

	principal.Sess.Nodes.Type = nodes.InstanceType
	principal.Sess.Nodes.Quantity = nodes.Instances
	principal.Sess.Nodes.Zones = nodes.Zones
	principal.Sess.Nodes.StorageSize = nodes.StorageSize
	principal.Sess.Nodes.StorageType = nodes.StorageType

	return conn.SaveState(principal.ID, principal.Sess)
}

func checkMachineType(t string) bool {
	for _, mType := range handledTypes {
		if t == mType {
			return true
		}
	}
	return false
}
