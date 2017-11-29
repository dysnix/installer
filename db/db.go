package db

import (
	"fmt"
	"time"

	"git.arilot.com/kuberstack/kuberstack-installer/savedstate"
)

var (
	errAlreadyExists = fmt.Errorf("Key already exists")
	errConnectClosed = fmt.Errorf("Connect closed")
	// errBadRecord     = fmt.Errorf("Record damaged")
)

// Connect interface represents a database connection with al the necessary methods
type Connect interface {
	SaveState(string, *savedstate.State) error
	Close() error
	String() string
	InsertState(string) error
	GetState(string) (*savedstate.State, error)
	Cleanup() ([][]byte, error)
}

// Open creates a new DB connection
func Open(driverName, dataSourceName string, ttl time.Duration) (Connect, error) {
	switch driverName {
	case "bolt":
		return newBoltDB(dataSourceName, ttl)
	default:
		panic(fmt.Errorf("Unsupported database type %q", driverName))
	}
}

// CleanupLoop used to call Connect.Cleanup() periodically until Connect wil be closed.
func CleanupLoop(logger func(string, ...interface{}), conn Connect, interval time.Duration) {
	ticker := time.NewTicker(time.Second / 10)
	var nextRun time.Time
	for range ticker.C {
		if time.Now().Before(nextRun) {
			continue
		}

		removed, err := conn.Cleanup()

		switch err {
		case nil:
			// do nothing, get out of switch
		case errConnectClosed:
			return
		default:
			panic(err)
		}

		for _, id := range removed {
			logger("ID expired: %q", id)
		}

		nextRun = time.Now().Add(interval)
	}
}
