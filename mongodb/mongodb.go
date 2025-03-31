// Package mongodb provides a service for interacting with MongoDB.
package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/operationspark/service-signup/greenlight"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongodbService struct {
	dbName string
	client *mongo.Client
}

func New(dbName string, client *mongo.Client) *MongodbService {
	return &MongodbService{
		dbName: dbName,
		client: client,
	}
}

// CreateUserJoinCode creates a join code document (including SessionID and ExpiresAt) and saves it to the Greenlight database.
// It returns the join code document ID and the session join code.
func (m *MongodbService) CreateUserJoinCode(ctx context.Context, sessionID string) (string, string, error) {
	userJoinCodeColl := m.client.Database(m.dbName).Collection("userJoinCodes")
	sessionColl := m.client.Database(m.dbName).Collection("sessions")

	s := sessionColl.FindOne(ctx, bson.M{"_id": sessionID}, &options.FindOneOptions{Projection: bson.M{"times": 1, "code": 1}})
	if s.Err() != nil {
		return "", "", fmt.Errorf("findOne: %w", s.Err())
	}

	var session greenlight.Session
	if err := s.Decode(&session); err != nil {
		return "", "", fmt.Errorf("decode session: %w", err)
	}

	joinData := greenlight.UserJoinCode{
		ExpiresAt: session.Times.Start.DateTime.Add(time.Hour * 8),
		SessionID: session.ID,
	}

	ior, err := userJoinCodeColl.InsertOne(ctx, joinData)
	if err != nil {
		return "", "", fmt.Errorf("insertOne: %w", err)
	}

	joinDataID := ior.InsertedID
	id := joinDataID.(primitive.ObjectID).Hex()

	return id, session.JoinCode, nil
}
