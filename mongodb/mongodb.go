package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/operationspark/service-signup/greenlight"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

// Creates a join code document (including SessionID and ExpiresAt) and saves it to the Greenlight database.
// Returns the join code document ID.
func (m *MongodbService) Create(ctx context.Context, sessionID string) (string, error) {
	userJoinCodeColl := m.client.Database(m.dbName).Collection("userJoinCode")
	sessionColl := m.client.Database(m.dbName).Collection("sessions")

	s := sessionColl.FindOne(ctx, bson.M{"_id": sessionID})
	if s.Err() != nil {
		return "", fmt.Errorf("findOne: %w", s.Err())
	}

	var session greenlight.Session
	if err := s.Decode(&session); err != nil {
		return "", fmt.Errorf("decode session: %w", err)
	}

	joinData := greenlight.UserJoinCode{
		ExpiresAt: session.Times.Start.DateTime.Add(time.Hour * 8),
		SessionID: session.ID,
	}

	ior, err := userJoinCodeColl.InsertOne(ctx, joinData)
	if err != nil {
		return "", fmt.Errorf("insertOne: %w", err)
	}

	joinDataID := ior.InsertedID
	id := joinDataID.(primitive.ObjectID).Hex()

	return id, nil
}
