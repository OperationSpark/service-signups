package mongodb_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var dbClient *mongo.Client
var dbName = "mongo-test"

// TestMain will only run if the build tags include "integration". This TestMain will setup a mongoDB instance with dockertest and tear the container down when finished. This setup/teardown process can take 40 seconds or more.
func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// Get Docker Hub credentials from environment variables
	dockerUsername := os.Getenv("DOCKER_USERNAME")
	dockerPassword := os.Getenv("DOCKER_PASSWORD")

	// pull mongodb docker image with authentication if credentials are provided
	runOptions := &dockertest.RunOptions{
		Repository: "mongo",
		Tag:        "7.0.18",
		Env: []string{
			// username and password for mongodb superuser
			"MONGO_INITDB_ROOT_USERNAME=root",
			"MONGO_INITDB_ROOT_PASSWORD=password",
		},
	}

	// Add Docker Hub authentication if credentials are provided
	if dockerUsername != "" && dockerPassword != "" {
		runOptions.Auth = docker.AuthConfiguration{
			Username: dockerUsername,
			Password: dockerPassword,
		}
	}

	resource, err := pool.RunWithOptions(
		runOptions,
		func(config *docker.HostConfig) {
			// set AutoRemove to true so that stopped container goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{
				Name: "no",
			}
		})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		var err error
		dbURL := fmt.Sprintf("mongodb://root:password@localhost:%s", resource.GetPort("27017/tcp"))
		dbClient, err = mongo.Connect(
			context.TODO(),
			options.Client().ApplyURI(dbURL),
		)
		if err != nil {
			return err
		}

		fmt.Printf("connected to DB @ %s\n", dbURL)
		return dbClient.Ping(context.TODO(), nil)
	})

	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// run tests
	code := m.Run()

	// When you're done, kill and remove the container
	if err = pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	// disconnect mongodb client
	if err = dbClient.Disconnect(context.TODO()); err != nil {
		panic(err)
	}

	os.Exit(code)
}
