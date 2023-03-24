package mongodb_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/operationspark/service-signup/greenlight"
	"github.com/operationspark/service-signup/mongodb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// test mongodb create function
func TestCreate(t *testing.T) {
	// Create a new MongoDB service for testing.
	srv := mongodb.New(dbName, dbClient)

	session := greenlight.Session{
		Cohort: "test",
	}

	var err error
	session.Times.Start.DateTime, err = time.Parse(time.RFC3339, "2023-01-02T15:00:00Z")
	require.NoError(t, err)

	s, err := dbClient.Database(dbName).Collection("sessions").InsertOne(context.Background(), &session)
	require.NoError(t, err)
	fmt.Printf("%+v\n", s)
	fmt.Printf("%+v\n", session)

	joinCodeId, err := srv.Create(context.Background(), session.ID)
	require.NoError(t, err)
	require.NotEmpty(t, joinCodeId)
	fmt.Print(joinCodeId)

	userJoinCodeColl := dbClient.Database(dbName).Collection("userJoinCode")

	joinCode := userJoinCodeColl.FindOne(context.Background(), bson.M{"ID": joinCodeId})
	require.NoError(t, joinCode.Err())

	var joinCodeDoc greenlight.UserJoinCode
	joinCode.Decode(&joinCodeDoc)

	require.Equal(t, session.ID, joinCodeDoc.ID)
	// require.Equal(t, )

}
