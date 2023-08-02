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
	LoadHeistStates() map[string]*Server
	SaveHeistState(*Server)
	LoadThemes() map[string]*Theme
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
	heistDir string
	themeDir string
}

// newFileStore creates a new file Store.
func newFileStore() Store {
	heistDir := os.Getenv("HEIST_FILE_STORE_DIR") + "heist/"
	themeDir := os.Getenv("HEIST_FILE_THEME_DIR")
	f := &fileStore{
		heistDir: heistDir,
		themeDir: themeDir,
	}
	return f
}

// SaveHeistState writes the heist state to the file system.
func (f *fileStore) SaveHeistState(server *Server) {
	data, err := json.Marshal(server)
	if err != nil {
		log.Error("Unable to marshal server "+server.ID+", error:", err)
		return
	}
	filename := f.heistDir + server.ID + ".json"
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		log.Error("Unable to save the Heist state for server "+server.ID+", error:", err)
	}
}

// LoadHeistStates reads the heist state from the file system. If the state
// cannot be found on the file system, then a new state is returned.
func (f *fileStore) LoadHeistStates() map[string]*Server {
	servers := make(map[string]*Server)

	files, err := os.ReadDir(f.heistDir)
	if err != nil {
		log.Warning("Failed to get the list of heist server json files, error:", err)
		return servers
	}

	for _, file := range files {
		filename := f.heistDir + file.Name()
		data, err := os.ReadFile(filename)
		if err != nil {
			log.Warning("Failed to read the data from file "+file.Name()+", error:", err)
		}

		var server Server
		err = json.Unmarshal(data, &server)
		if err != nil {
			log.Error("Unable to unmarshal server data from file "+file.Name()+", error:", err)
		} else {
			servers[server.ID] = &server
		}
	}

	return servers
}

// LoadThemes loads all available themes that may be used with the heist bot.
func (f *fileStore) LoadThemes() map[string]*Theme {
	themes := make(map[string]*Theme)

	files, err := os.ReadDir(f.themeDir)
	if err != nil {
		log.Warning("Failed to get the list of heist server json files, error:", err)
		return themes
	}

	for _, file := range files {
		filename := f.themeDir + file.Name()
		data, err := os.ReadFile(filename)
		if err != nil {
			log.Warning("Failed to read the data from file "+file.Name()+", error:", err)
		}

		var theme Theme
		err = json.Unmarshal(data, &theme)
		if err != nil {
			log.Error("Unable to unmarshal server data from file "+file.Name()+", error:", err)
		} else {
			themes[theme.ID] = &theme
		}
	}

	return themes
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
func (m *mongodb) SaveHeistState(server *Server) {
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
			log.Error("Failed to create the heist collection, error:", err)
			return
		}
		heistCollection = heistDB.Collection("heist")
	}

	_, err = heistCollection.InsertOne(ctx, server)
	if err != nil {
		_, err = heistCollection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: server.ID}}, server)
		if err != nil {
			log.Error("failed to update or insert "+server.ID+"into heist collection, error:", err)
		}
	}
}

// LoadHeistStates loads the heist state from the MongoDB database.
func (m *mongodb) LoadHeistStates() map[string]*Server {
	log.Debug("--> LoadHeistState")
	defer log.Debug("<-- LoadHeistState")

	servers = make(map[string]*Server)

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
		return servers
	}
	defer client.Disconnect(ctx)
	defer log.Debug("Disconnected from DB")

	heistDB := client.Database(m.dbName)
	heistCollection := heistDB.Collection("heist")
	if heistCollection == nil {
		log.Error("Failed to create the heist collection, error:", err)
		return servers
	}
	// defer heistCollection.Drop(ctx) // Use this if you want to get rid of the collection once done
	log.Debug("Collection:", heistCollection.Name())

	cur, err := heistCollection.Find(ctx, bson.D{{}})
	if err != nil {
		log.Error("Unable to get a cursor into the heist collection, error:", err)
		return servers
	}

	for cur.Next(ctx) {
		var server Server
		err = cur.Decode(&server)
		if err != nil {
			log.Error("Failed to decode server, error:", err)
			continue
		}
		servers[server.ID] = &server
	}

	return servers
}

// LoadThemes loads all available themes that may be used with the heist bot.
func (m *mongodb) LoadThemes() map[string]*Theme {
	log.Debug("--> LoadHeistState")
	defer log.Debug("<-- LoadHeistState")

	themes = make(map[string]*Theme)

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
		return themes
	}
	defer client.Disconnect(ctx)
	defer log.Debug("Disconnected from DB")

	heistDB := client.Database(m.dbName)
	heistCollection := heistDB.Collection("theme")
	if heistCollection == nil {
		log.Error("Failed to create the theme collection, error:", err)
		return themes
	}
	// defer heistCollection.Drop(ctx)
	log.Debug("Collection:", heistCollection.Name())

	cur, err := heistCollection.Find(ctx, bson.D{{}})
	if err != nil {
		log.Error("Unable to get a cursor into the theme collection, error:", err)
		return themes
	}

	for cur.Next(ctx) {
		var theme Theme
		err = cur.Decode(&theme)
		if err != nil {
			log.Error("Failed to decode theme, error:", err)
			continue
		}
		themes[theme.ID] = &theme
	}

	return themes
}
