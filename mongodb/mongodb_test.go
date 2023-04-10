package mongodb_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/operationspark/service-signup/greenlight"
	"github.com/operationspark/service-signup/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// test mongodb create function
func TestCreate(t *testing.T) {
	// Create a new MongoDB service for testing.
	srv := mongodb.New(dbName, dbClient)

	session := greenlight.Session{
		ID:       randID(), // Simulate Greenlight IDs (not using ObjectId())
		Cohort:   "test",
		JoinCode: "tlav",
	}

	var err error
	session.Times.Start.DateTime, err = time.Parse(time.RFC3339, "2023-01-02T15:00:00Z")
	require.NoError(t, err)

	_, err = dbClient.Database(dbName).Collection("sessions").InsertOne(context.Background(), session)
	require.NoError(t, err)

	joinCodeId, sessionJoinCode, err := srv.Create(context.Background(), session.ID)
	require.NoError(t, err)
	require.NotEmpty(t, joinCodeId)

	userJoinCodeColl := dbClient.Database(dbName).Collection("userJoinCodes")

	objID, err := primitive.ObjectIDFromHex(joinCodeId) // change string ID to Mongo ObjectID
	require.NoError(t, err)

	joinCodeDoc := userJoinCodeColl.FindOne(context.Background(), bson.M{"_id": objID})
	require.NoError(t, joinCodeDoc.Err())

	var joinCode greenlight.UserJoinCode
	err = joinCodeDoc.Decode(&joinCode)
	require.NoError(t, err)

	wantExpiresAt, err := time.Parse(time.RFC3339, "2023-01-02T23:00:00Z")
	require.NoError(t, err)

	require.Equal(t, wantExpiresAt, joinCode.ExpiresAt)
	require.Equal(t, session.JoinCode, sessionJoinCode)

}

// RandID generates a random 17-character string to simulate Meteor's Mongo ID generation.
// Meteor did not originally use Mongo's ObjectID() for document IDs.
func randID() string {
	var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	length := 17
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
