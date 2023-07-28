package economy

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// bankStore defines the methods required to load and save the economy state.
type bankStore interface {
	loadBanks() map[string]*Bank
	saveBank(*Bank)
}

// NewStore creates a new store to be used to load and save the economy state.
func newStore() bankStore {
	storeType := os.Getenv("HEIST_STORE")
	log.Debug("Storage type:", storeType)
	var store bankStore
	if storeType == "file" {
		store = newFileStore()
	} else {
		store = newMongoStore()
	}
	return store
}

// fileStore is a Store used to load and save the economy state to a file.
type fileStore struct {
	economyDir string
}

// newFileStore creates a new file Store.
func newFileStore() bankStore {
	heistDir := os.Getenv("HEIST_FILE_STORE_DIR") + "economy/"
	f := &fileStore{
		economyDir: heistDir,
	}
	return f
}

// saveBank writes the bank state to the file system.
func (f *fileStore) saveBank(bank *Bank) {
	data, err := json.Marshal(bank)
	if err != nil {
		log.Error("Unable to marshal bank "+bank.ID+", error:", err)
		return
	}
	filename := f.economyDir + bank.ID + ".json"
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		log.Error("Unable to save the bank state for server "+bank.ID+", error:", err)
	}
}

// loadBanks reads the bank states from the file system.
func (f *fileStore) loadBanks() map[string]*Bank {
	banks := make(map[string]*Bank)

	files, err := os.ReadDir(f.economyDir)
	if err != nil {
		log.Warning("Failed to get the list of banks json files, error:", err)
		return banks
	}

	for _, file := range files {
		filename := f.economyDir + file.Name()
		data, err := os.ReadFile(filename)
		log.Debug("Loading economy " + filename)
		if err != nil {
			log.Warning("Failed to read the data from file "+file.Name()+", error:", err)
		}

		var bank Bank
		err = json.Unmarshal(data, &bank)
		if err != nil {
			log.Error("Unable to unmarshal bank data from file "+file.Name()+", error:", err)
		} else {
			banks[bank.ID] = &bank
		}
	}

	return banks
}

// mongodb is a Store used to load and save the economy state to a MongoDB database.
type mongodb struct {
	adminDB string
	dbName  string
	pwd     string
	uri     string
	userID  string
}

// newMongoStore creates a Store to load and save the economy state to a MongoDB database.
func newMongoStore() bankStore {
	godotenv.Load()

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("You must set your 'MONGODB_URI' environmental variable. See\n\t https://www.mongodb.com/docs/drivers/go/current/usage-examples/#environment-variable")
	}
	dbName := os.Getenv("MONGODB_DATABASE")
	if dbName == "" {
		log.Fatal("You must set your 'MONGODB_DATABASE' environment variable")
	}
	userID := os.Getenv("MONGODB_USERID")
	if userID == "" {
		log.Fatal("You must set your 'MONGODB_USERID' environment variable")
	}
	pwd := os.Getenv("MONGODB_PASSWORD")
	if pwd == "" {
		log.Fatal("You must set your 'MONGODB_PASSWORD' environment variable")
	}
	adminDB := os.Getenv("MONGODB_ADMIN_DB")
	if adminDB == "" {
		adminDB = "admin"
	}

	m := mongodb{
		adminDB: adminDB,
		dbName:  dbName,
		pwd:     pwd,
		uri:     uri,
		userID:  userID,
	}

	// Wait for MongoDB to become active before proceeding
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	credential := options.Credential{
		AuthSource: m.adminDB,
		Username:   m.userID,
		Password:   m.pwd,
	}
	clientOpts := options.Client().ApplyURI(m.uri).SetAuth(credential)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatal("Unable to connect to the MongoDB database, error:", err)
		return nil
	}
	defer client.Disconnect(ctx)
	// Check the connection
	err = client.Ping(ctx, nil)

	if err != nil {
		log.Fatal("Unable to ping the MongoDB database, error:", err)
	}

	return &m
}

// saveBank stores the bank state in the MongoDB database.
func (m *mongodb) saveBank(bank *Bank) {
	log.Debug("--> SaveBank")
	defer log.Debug("<-- SaveBank")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	credential := options.Credential{
		AuthSource: m.adminDB,
		Username:   m.userID,
		Password:   m.pwd,
	}
	clientOpts := options.Client().ApplyURI(m.uri).SetAuth(credential)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Error("Unable to connect to the MongoDB database, error:", err)
		return
	}
	defer client.Disconnect(ctx)
	findOptions := options.Find()
	//Set the limit of the number of record to find
	findOptions.SetLimit(5)
	defer log.Debug("Disconnected from DB")

	heistDB := client.Database(m.dbName)
	heistCollection := heistDB.Collection("economy")
	if heistCollection == nil {
		if err = heistDB.CreateCollection(ctx, "economy"); err != nil {
			log.Error("Failed to create the heist collection, error:", err)
			return
		}
		heistCollection = heistDB.Collection("economy")
	}

	_, err = heistCollection.InsertOne(ctx, bank)
	if err != nil {
		_, err = heistCollection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: bank.ID}}, bank)
		if err != nil {
			log.Error("failed to update or insert "+bank.ID+"into heist collection, error:", err)
		}
	}
}

// loadBanks loads the economy state from the MongoDB database.
func (m *mongodb) loadBanks() map[string]*Bank {
	log.Debug("--> LoadBanks")
	defer log.Debug("<-- LoadBanks")

	banks := make(map[string]*Bank)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	credential := options.Credential{
		AuthSource: m.adminDB,
		Username:   m.userID,
		Password:   m.pwd,
	}
	clientOpts := options.Client().ApplyURI(m.uri).SetAuth(credential)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Error("Unable to connect to the MongoDB database, error:", err)
		return banks
	}
	defer client.Disconnect(ctx)
	defer log.Debug("Disconnected from DB")

	heistDB := client.Database(m.dbName)
	heistCollection := heistDB.Collection("economy")
	if heistCollection == nil {
		log.Error("Failed to create the bankn collection, error:", err)
		return banks
	}
	// defer heistCollection.Drop(ctx)
	log.Debug("Collection:", heistCollection.Name())

	cur, err := heistCollection.Find(ctx, bson.D{{}})
	if err != nil {
		log.Error("Unable to get a cursor into the heist collection, error:", err)
		return banks
	}

	for cur.Next(ctx) {
		var bank Bank
		err = cur.Decode(&bank)
		if err != nil {
			log.Error("Failed to decode bank, error:", err)
			continue
		}
		banks[bank.ID] = &bank
	}

	return banks
}
