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
	//adminDB string
	//dbName  string
	//pwd     string
	uri string
	//userID  string
}

var clientOpts *options.ClientOptions = nil
var client *mongo.Client = nil
var err error = nil

// newMongoStore creates a Store to load and save documents in a MongoDB database.
func newMongoStore() StoreInterface {
	godotenv.Load()

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("You must set your 'MONGODB_URI' environmental variable. See\n\t https://www.mongodb.com/docs/drivers/go/current/usage-examples/#environment-variable")
	}
	//dbName := os.Getenv("MONGODB_DATABASE")
	//if dbName == "" {
	//	log.Fatal("You must set your 'MONGODB_DATABASE' environment variable")
	//}
	//userID := os.Getenv("MONGODB_USERID")
	//if userID == "" {
	//	log.Fatal("You must set your 'MONGODB_USERID' environment variable")
	//}
	//pwd := os.Getenv("MONGODB_PASSWORD")
	//if pwd == "" {
	//	log.Fatal("You must set your 'MONGODB_PASSWORD' environment variable")
	//}
	//adminDB := os.Getenv("MONGODB_ADMIN_DB")
	//if adminDB == "" {
	//	adminDB = "admin"
	//}

	m := mongodb{
		//	adminDB: adminDB,
		//	dbName:  dbName,
		//	pwd:     pwd,
		uri: uri,
		//	userID:  userID,
	}

	// Wait for MongoDB to become active before proceeding
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//credential := options.Credential{
	//	AuthSource: m.adminDB,
	//	Username:   m.userID,
	//	Password:   m.pwd,
	//}
	if clientOpts == nil {
		clientOpts = options.Client().ApplyURI(m.uri) //.SetAuth(credential)
		client, err = mongo.Connect(ctx, clientOpts)
	}

	if err != nil {
		log.Fatal("Unable to connect to the MongoDB database, error:", err)
		err = nil
		return nil
	}
	//defer client.Disconnect(ctx)
	// Check the connection
	err = client.Ping(ctx, nil)

	if err != nil {
		log.Fatal("Unable to ping the MongoDB database, error:", err)
		err = nil
	}

	return &m
}

// ListDocuments returns the ID of each document in a collection in the collection.
func (m *mongodb) ListDocuments(collectionName string) []string {
	log.Trace("--> ListDocuments")
	defer log.Trace("<-- ListDocuments")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//credential := options.Credential{
	//	AuthSource: m.adminDB,
	//	Username:   m.userID,
	//	Password:   m.pwd,
	//}
	if clientOpts == nil {
		clientOpts = options.Client().ApplyURI(m.uri) //.SetAuth(credential)
		client, err = mongo.Connect(ctx, clientOpts)
	}
	if err != nil {
		log.Error("Unable to connect to the MongoDB database, error:", err)
		err = nil
		return nil
	}
	//defer client.Disconnect(ctx)

	db := client.Database("Heist")
	collection := db.Collection(collectionName)
	if collection == nil {
		log.Errorf("Failed to create %s collection, error=%s", collectionName, err.Error())
		return nil
	}
	opts := options.Find().SetProjection(bson.M{"_id": 1})
	cur, err := collection.Find(ctx, bson.D{}, opts)
	if err != nil {
		log.Errorf("Failed to search the %s collection, error=%s", collectionName, err.Error())
		err = nil
		return nil
	}
	type result struct {
		ID string `bson:"_id"`
	}
	var results []result

	err = cur.All(ctx, &results)
	if err != nil {
		log.Errorf("Error getting the IDs for collection %s, error=%s", collectionName, err.Error())
		err = nil
	}

	idList := make([]string, 0, len(results))
	for _, r := range results {
		idList = append(idList, r.ID)
	}

	return idList
}

// Load loads a document identified by documentID from the collection into data.
func (m *mongodb) Load(collectionName string, documentID string, data interface{}) {
	log.Trace("--> Load")
	defer log.Trace("<-- Load")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//credential := options.Credential{
	//	AuthSource: m.adminDB,
	//	Username:   m.userID,
	//	Password:   m.pwd,
	//}
	if clientOpts == nil {
		clientOpts = options.Client().ApplyURI(m.uri) //.SetAuth(credential)
		client, err = mongo.Connect(ctx, clientOpts)
	}
	if err != nil {
		log.Error("Unable to connect to the MongoDB database, error:", err)
		err = nil
		return
	}
	//defer client.Disconnect(ctx)

	db := client.Database("Heist")
	collection := db.Collection(collectionName)
	if collection == nil {
		log.Errorf("Failed to create %s collection, error=%s", collectionName, err.Error())
		err = nil
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
		err = nil
	}
}

// Save stores data into a documeent within the specified collection.
func (m *mongodb) Save(collectionName string, documentID string, data interface{}) {
	log.Trace("--> Save")
	defer log.Trace("<-- Save")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//credential := options.Credential{
	//	AuthSource: m.adminDB,
	//	Username:   m.userID,
	//	Password:   m.pwd,
	//}
	if clientOpts == nil {
		clientOpts = options.Client().ApplyURI(m.uri) //.SetAuth(credential)
		client, err = mongo.Connect(ctx, clientOpts)
	}
	if err != nil {
		log.Error("Unable to connect to the MongoDB database, error:", err)
		err = nil
		return
	}
	//defer client.Disconnect(ctx)
	findOptions := options.Find()
	//Set the limit of the number of record to find
	findOptions.SetLimit(5)

	db := client.Database("Heist")
	collection := db.Collection(collectionName)
	if collection == nil {
		if err = db.CreateCollection(ctx, collectionName); err != nil {
			log.Errorf("Failed to create the %s collection, error=%s", collectionName, err.Error())
			err = nil
			return
		}
		collection = db.Collection(collectionName)
	}

	_, err = collection.InsertOne(ctx, data)
	if err != nil {
		_, err = collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: documentID}}, data)
		if err != nil {
			log.Errorf("Failed to update or insert document %s into %s collection, error=%s", documentID, collectionName, err.Error())
			err = nil
		}
	}
}
