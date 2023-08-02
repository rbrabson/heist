package store

import (
	"os"

	log "github.com/sirupsen/logrus"
)

// Store defines the methods required to load and save the heist state.
type Store interface {
	ListDocuments(collection string) []string
	Load(collection string, docuentID string, data interface{})
	Save(collection string, documentID string, data interface{})
}

// NewStore creates a new store to be used to load and save the heist state.
func NewStore() Store {
	storeType := os.Getenv("HEIST_STORE")
	log.Debug("Storage type:", storeType)
	var store Store
	if storeType == "file" {
		store = newFileStore()
	} else {
		store = newMongoStore()
	}
	return store
}
