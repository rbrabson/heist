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

func (f *fileStore) LoadHeistState() ([]byte, error) {
	data, err := os.ReadFile(f.fileName)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (f *fileStore) SaveEnconomyState(economy.Banks) {
	// TODO
}
