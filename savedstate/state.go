package savedstate

import "time"

// NodesParams are the parameters for the group of nodes
type NodesParams struct {
	Type        string
	Quantity    int64
	Zones       []string
	StorageSize int64
	StorageType string
}

// State data struct as it will be stored in DB
type State struct {
	Ctime  time.Time
	Mtime  time.Time
	Expire time.Time

	AccessKey string
	Region    string
	SecretKey string
	SSHPubKey string

	Domain      string
	Name        string
	Type        int64
	ZoneID      string
	ZoneWatchID string
	RecWatchID  string
	Bucket      string

	Master NodesParams
	Nodes  NodesParams

	Products []string

	Kubecfg []byte
}
