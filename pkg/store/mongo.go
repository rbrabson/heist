package store

import (
	"context"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	log "github.com/sirupsen/logrus"
)

// mongodb is a Store used to load and save documents in a MongoDB database.
type mongodb struct {
	adminDB string
	dbName  string
	pwd     string
	uri     string
	userID  string
}

// newMongoStore creates a Store to load and save documents in a MongoDB database.
func newMongoStore() StoreInterface {
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

// ListDocuments returns the ID of each document in a collection in the collection.
func (m *mongodb) ListDocuments(collectionName string) []string {
	log.Debug("--> ListDocuments")
	defer log.Debug("<-- ListDocuments")

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
		return nil
	}
	defer client.Disconnect(ctx)
	defer log.Debug("Disconnected from DB")

	db := client.Database(m.dbName)
	collection := db.Collection(collectionName)
	if collection == nil {
		log.Errorf("Failed to create %s collection, error=%s", collectionName, err.Error())
		return nil
	}
	opts := options.Find().SetProjection(bson.M{"_id": 1})
	cur, err := collection.Find(ctx, bson.D{}, opts)
	if err != nil {
		log.Errorf("Failed to search the %s collection, error=%s", collectionName, err.Error())
		return nil
	}
	type result struct {
		ID string `bson:"_id"`
	}
	var results []result

	err = cur.All(ctx, &results)
	if err != nil {
		log.Errorf("Error getting the IDs for collection %s, error=%s", collectionName, err.Error())
	}

	idList := make([]string, 0, len(results))
	for _, r := range results {
		idList = append(idList, r.ID)
	}

	return idList
}

// Load loads a document identified by documentID from the collection into data.
func (m *mongodb) Load(collectionName string, documentID string, data interface{}) {
	log.Debug("--> Load")
	defer log.Debug("<-- Load")

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
	defer log.Debug("Disconnected from DB")

	db := client.Database(m.dbName)
	collection := db.Collection(collectionName)
	if collection == nil {
		log.Errorf("Failed to create %s collection, error=%s", collectionName, err.Error())
		return
	}
	log.Debug("Collection:", collection.Name())

	res := collection.FindOne(ctx, bson.D{{Key: "_id", Value: documentID}})
	if res == nil {
		log.Errorf("Unable to find document %s in collection %s", documentID, collectionName)
	}
	err = res.Decode(data)
	if err != nil {
		log.Errorf("Failed to decode document %s, error:%s", documentID, err.Error())
	}
}

// Save stores data into a documeent within the specified collection.
func (m *mongodb) Save(collectionName string, documentID string, data interface{}) {
	log.Debug("--> Save")
	defer log.Debug("<-- Save")

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

	db := client.Database(m.dbName)
	collection := db.Collection(collectionName)
	if collection == nil {
		if err = db.CreateCollection(ctx, collectionName); err != nil {
			log.Errorf("Failed to create the %s collection, error=%s", collectionName, err.Error())
			return
		}
		collection = db.Collection(collectionName)
	}

	_, err = collection.InsertOne(ctx, data)
	if err != nil {
		_, err = collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: documentID}}, data)
		if err != nil {
			log.Errorf("Failed to update or insert document %s into %s collection, error=%s", documentID, collectionName, err.Error())
		}
	}
}
