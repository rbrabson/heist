package store

import (
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

var (
	// The store used by all bots to list, load and store documents
	Store StoreInterface
)

// Initializes the store used by all bots
func init() {
	godotenv.Load()
	Store = newStore()
}

// StoreInterface defines the methods required to load and save the heist state.
type StoreInterface interface {
	ListDocuments(collection string) []string
	Load(collection string, docuentID string, data interface{})
	Save(collection string, documentID string, data interface{})
}

// newStore creates a new store to be used to load and save the heist state.
func newStore() StoreInterface {
	storeType := os.Getenv("HEIST_STORE")
	log.Debug("Storage type:", storeType)
	var store StoreInterface
	if storeType == "file" {
		store = newFileStore()
	} else {
		store = newMongoStore()
	}
	return store
}
