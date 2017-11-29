package db

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"git.arilot.com/kuberstack/kuberstack-installer/savedstate"

	"github.com/boltdb/bolt"
)

var (
	savedstatesBucket = []byte("savedstates")
)

type boltDB struct {
	db     *bolt.DB
	ttl    time.Duration
	closed bool

	sync.RWMutex
}

func newBoltDB(filePath string, ttl time.Duration) (Connect, error) {
	db, err := bolt.Open(filePath, 0640, nil)
	if err != nil {
		return nil, err
	}

	return &boltDB{db: db, ttl: ttl}, nil
}

func (conn *boltDB) SaveState(id string, content *savedstate.State) error {
	conn.RLock()
	defer conn.RUnlock()

	content.Mtime = time.Now()
	content.Expire = content.Mtime.Add(conn.ttl)

	value, err := json.Marshal(&content)
	if err != nil {
		panic(err)
	}

	return conn.db.Update(
		func(tx *bolt.Tx) error {
			return updateTransaction(tx, []byte(id), value)
		},
	)
}

func (conn *boltDB) Close() error {
	conn.Lock()
	defer conn.Unlock()

	conn.closed = true
	return conn.db.Close()
}

func (conn *boltDB) String() string {
	return conn.db.String()
}

func (conn *boltDB) Cleanup() ([][]byte, error) {
	conn.RLock()
	defer conn.RUnlock()

	if conn.closed {
		return nil, errConnectClosed
	}

	toRemove, err := countExpired(conn.db)
	if err != nil {
		return nil, err
	}

	err = deleteExpired(conn.db, toRemove)
	if err != nil {
		return nil, err
	}

	return toRemove, nil
}

func (conn *boltDB) GetState(id string) (*savedstate.State, error) {
	conn.RLock()
	defer conn.RUnlock()

	key := []byte(id)

	tx, err := conn.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer closeRO(tx)

	bucket := tx.Bucket(savedstatesBucket)
	if bucket == nil {
		return nil, nil
	}

	exists := bucket.Get(key)
	if exists == nil {
		return nil, nil
	}

	content, err := unmarshalsavedstate(exists)
	if err != nil {
		return nil, err
	}

	if content.Expire.Before(time.Now()) {
		return nil, nil
	}

	return content, nil
}

func (conn *boltDB) InsertState(id string) error {
	conn.RLock()
	defer conn.RUnlock()

	content := &savedstate.State{
		Ctime:  time.Now(),
		Mtime:  time.Now(),
		Expire: time.Now().Add(conn.ttl),
	}

	value, err := json.Marshal(&content)
	if err != nil {
		panic(err)
	}

	return conn.db.Update(
		func(tx *bolt.Tx) error {
			return insertTransaction(tx, []byte(id), value)
		},
	)
}

func insertTransaction(tx *bolt.Tx, key []byte, value []byte) error {
	bucket, err := tx.CreateBucketIfNotExists(savedstatesBucket)
	if err != nil {
		return err
	}

	exists := bucket.Get(key)

	if exists == nil {
		return bucket.Put(key, value)
	}

	content := savedstate.State{}

	err = json.Unmarshal(exists, &content)
	if err != nil {
		return err
	}

	if content.Expire.After(time.Now()) {
		return errAlreadyExists
	}

	return bucket.Put(key, value)
}

func updateTransaction(tx *bolt.Tx, key []byte, value []byte) error {
	bucket := tx.Bucket(savedstatesBucket)
	if bucket == nil {
		return fmt.Errorf("Bucket does not exists: %q", savedstatesBucket)
	}

	return bucket.Put(key, value)
}

func closeRO(tx *bolt.Tx) {
	err := tx.Rollback()
	if err != nil {
		panic(fmt.Errorf("Error closing RO transaction: %v", err))
	}
}

func unmarshalsavedstate(data []byte) (*savedstate.State, error) {
	content := savedstate.State{}

	err := json.Unmarshal(data, &content)
	if err != nil {
		return nil, err
	}

	return &content, nil
}

func countExpired(conn *bolt.DB) ([][]byte, error) {
	toRemove := make([][]byte, 0, 100)

	tx, err := conn.Begin(false)
	if err != nil {
		return nil, err
	}
	defer closeRO(tx)

	bucket := tx.Bucket(savedstatesBucket)
	if bucket == nil {
		return nil, nil
	}

	err = bucket.ForEach(
		func(key, data []byte) (err error) {
			content, err := unmarshalsavedstate(data)
			if err != nil || content.Expire.Before(time.Now()) {
				toRemove = append(toRemove, key)
			}
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	return toRemove, nil
}

func deleteExpired(conn *bolt.DB, toRemove [][]byte) (err error) {
	for _, key := range toRemove {
		err = conn.Update(
			func(tx *bolt.Tx) error {
				bucket := tx.Bucket(savedstatesBucket)
				if bucket == nil {
					return nil
				}
				return bucket.Delete(key)
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}
