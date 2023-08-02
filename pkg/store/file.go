package store

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// fileStore is a Store used to load and save a document to a file.
type fileStore struct {
	dir string
}

// newFileStore creates a new file Store.
func newFileStore() StoreInterface {
	dir := os.Getenv("HEIST_FILE_STORE_DIR")
	f := &fileStore{
		dir: dir,
	}
	return f
}

// ListDocuments returns the list of files in the sub-directory (collection).
func (f *fileStore) ListDocuments(collection string) []string {
	dirName := f.dir + "/" + collection
	files, err := os.ReadDir(dirName)
	if err != nil {
		log.Errorf("Failed to get the list of files for colledction %s, error=%s", collection, err.Error())
		return nil
	}
	fileNames := make([]string, 0, len(files))
	for _, file := range files {
		split := strings.Split(file.Name(), ".json")
		fileNames = append(fileNames, split[0])
	}
	return fileNames
}

// Load loads a file identified by documentID from the subdirectory (collection) into data.
func (f *fileStore) Load(collection string, documentID string, data interface{}) {
	log.Debug("--> Load")
	defer log.Debug("<-- Load")

	filename := fmt.Sprintf("%s%s/%s.json", f.dir, collection, documentID)
	b, err := os.ReadFile(filename)
	if err != nil {
		log.Errorf("Failed to read the data from file %s, error=%s", filename, err.Error())
		return
	}

	err = json.Unmarshal(b, data)
	if err != nil {
		log.Errorf("Unable to unmarshal data for collection %s, documentID %s, error=%s", collection, documentID, err.Error())
	}
}

// Save stores data into a subdirectory (collection) with the file name documentID.
func (f *fileStore) Save(collection string, documentID string, data interface{}) {
	log.Debug("--> Save")
	defer log.Debug("<-- Save")

	b, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Unable to marshal data for document %s, error=%s", collection, err.Error())
		return
	}

	filename := f.dir + collection + "/" + documentID + ".json"
	err = os.WriteFile(filename, b, 0644)
	if err != nil {
		log.Errorf("Unable to save the document, collection=%s, document=%s, error=%s", collection, documentID, err.Error())
	}
}
