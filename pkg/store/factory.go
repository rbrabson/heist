package store

import (
	"github.com/rbrabson/heist/pkg/economy"
)

var (
	storeType = "file"
)

type Store interface {
	SaveHeistState([]byte)
	SaveEnconomyState(economy.Banks)
}

func NewStore() Store {
	var store Store
	if storeType == "file" {
		store = newFileStore()
	} else {
		store = newMongoStore()
	}
	return store
}
