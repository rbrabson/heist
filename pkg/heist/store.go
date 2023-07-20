package heist

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

// Store defines the methods required to load and save the heist state.
type Store interface {
	LoadHeistState() *Servers
	SaveHeistState(*Servers)
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

// fileStore is a Store used to load and save the heist state to a file.
type fileStore struct {
	fileName string
}

// newFileStore creates a new file Store.
func newFileStore() Store {
	dir := os.Getenv("HEIST_FILE_STORE_DIR")
	filename := os.Getenv("HEIST_FILE_NAME")
	f := &fileStore{
		fileName: dir + filename,
	}
	return f
}

// SaveHeistState writes the heist state to the file system.
func (f *fileStore) SaveHeistState(servers *Servers) {
	data, err := json.MarshalIndent(servers, "", " ")
	if err != nil {
		log.Error("Unable to marshal servers, error:", err)
		return
	}
	err = os.WriteFile(f.fileName, data, 0644)
	if err != nil {
		log.Error("Unable to save the Heist state, error:", err)
	}
}

// LoadHeistState reads the heist state from the file system. If the state
// cannot be found on the file system, then a new state is returned.
func (f *fileStore) LoadHeistState() *Servers {
	data, err := os.ReadFile(f.fileName)
	if err != nil {
		return NewServers()
	}
	var servers Servers
	err = json.Unmarshal(data, &servers)
	if err != nil {
		log.Error("unable to unmarshal server data")
		return NewServers()
	}
	return &servers
}

// mongodb is a Store used to load and save the heist state to a MongoDB database.
type mongodb struct {
	adminDB string
	dbName  string
	pwd     string
	uri     string
	userID  string
}

// newMongoStore creates a Store to load and save the heiss state to a MongoDB database.
func newMongoStore() Store {
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

// SaveHeistState stores the heist state in the MongoDB database.
func (m *mongodb) SaveHeistState(servers *Servers) {
	log.Debug("--> SaveHeistState")
	defer log.Debug("<-- SaveHeistState")

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
	heistCollection := heistDB.Collection("heist")
	if heistCollection == nil {
		if err = heistDB.CreateCollection(ctx, "heist"); err != nil {
			log.Error("failed to create the heist collection, error:", err)
			return
		}
		heistCollection = heistDB.Collection("heist")
	}

	_, err = heistCollection.InsertOne(ctx, servers)
	if err != nil {
		_, err = heistCollection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: "heist"}}, servers)
		if err != nil {
			log.Error("failed to update or insert into heist collection, error:", err)
		}
	}
}

// LoadHeistState loads the heist state from the MongoDB database.
func (m *mongodb) LoadHeistState() *Servers {
	log.Debug("--> LoadHeistState")
	defer log.Debug("<-- LoadHeistState")

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
		return NewServers()
	}
	defer client.Disconnect(ctx)
	defer log.Debug("Disconnected from DB")

	heistDB := client.Database(m.dbName)
	heistCollection := heistDB.Collection("heist")
	if heistCollection == nil {
		log.Error("Failed to create the heist collection, error:", err)
		return NewServers()
	}
	// defer heistCollection.Drop(ctx)
	log.Debug("Collection:", heistCollection.Name())

	result := heistCollection.FindOne(ctx, bson.D{{Key: "_id", Value: "heist"}})
	var servers Servers
	err = result.Decode(&servers)
	if err != nil {
		log.Error("Failed to decode servers, error:", err)
		return NewServers()
	}

	return &servers
}
