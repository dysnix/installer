// Package auth is a package holding step 0 handlers and structures
package auth

import (
	"git.arilot.com/kuberstack/kuberstack-installer/db"
	"git.arilot.com/kuberstack/kuberstack-installer/savedstate"
	uuid "github.com/satori/go.uuid"
)

// GetSession checks the auth token for the restapi calls
func GetSession(db db.Connect, token string) (*savedstate.Principal, error) {
	content, err := db.GetState(token)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}
	return &savedstate.Principal{ID: token, Sess: content}, nil
}

// GetSessionID generates an token for the restapi calls
func GetSessionID(db db.Connect) (string, error) {
	id := uuid.NewV4().String()

	err := db.InsertState(id)
	if err != nil {
		return "", err
	}

	return id, nil
}
