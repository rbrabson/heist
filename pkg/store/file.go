package store

import (
	"log"
	"os"

	"github.com/rbrabson/heist/pkg/economy"
)

type fileStore struct {
	fileName string
}

func newFileStore() Store {
	f := &fileStore{
		fileName: "./store/heist/servers.json",
	}
	return f
}

func (f *fileStore) SaveHeistState(data []byte) {
	err := os.WriteFile(f.fileName, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func (f *fileStore) SaveEnconomyState(economy.Banks) {

}
