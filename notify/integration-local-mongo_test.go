//go:build !integration
// +build !integration

package notify

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var dbClient *mongo.Client
var dbName = "notify-test"

// TestMain will only run if the build tags do not contain "integration". This TestMain will setup a connection to a local database and runs much faster than the dockertest setup.
func TestMain(m *testing.M) {
	mongoURI := os.Getenv("MONGO_URI")
	if len(mongoURI) == 0 {
		mongoURI = fmt.Sprintf("mongodb://localhost:27017/%s", dbName)
	}

	client, err := mongo.Connect(
		context.TODO(),
		options.Client().ApplyURI(mongoURI).SetConnectTimeout(time.Second*5),
	)
	if err != nil {
		panic(fmt.Errorf("connect: %v", err))
	}

	dbClient = client

	err = client.Database(dbName).Drop(context.Background())
	if err != nil {
		panic(fmt.Errorf("drop database %q: %v", dbName, err))
	}

	// Run the tests
	code := m.Run()

	// disconnect mongodb client
	if err = dbClient.Disconnect(context.TODO()); err != nil {
		panic(err)
	}

	os.Exit(code)
}
