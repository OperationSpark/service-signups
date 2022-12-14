package notify

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGetUpcomingSessions(t *testing.T) {
	uri := fmt.Sprintf("mongodb://localhost:27017/%s", dbName)

	t.Run("Only Retrieves sessions in the given time frame", func(t *testing.T) {
		mSrv, err := NewMongoService(context.Background(), uri)
		require.NoError(t, err)

		dropDatabase(context.Background(), mSrv)

		// Insert 10 sessions over the next 10 days.
		for daysInFuture := 1; daysInFuture <= 10; daysInFuture++ {
			insertFutureSession(t, mSrv, daysInFuture)
		}

		daysInFuture := 7
		sessions, err := mSrv.getUpcomingSessions(context.Background(), daysInFuture)
		require.NoError(t, err)

		require.WithinRange(
			t,
			sessions[0].Times.Start.DateTime,
			time.Now(),
			time.Now().AddDate(0, 0, daysInFuture),
		)

		require.Len(t, sessions, 7, "should find the a session for each of the next 7 days")
	})

	t.Run("session responses contain the expected attendees of the given session", func(t *testing.T) {
		mSrv, err := NewMongoService(context.Background(), uri)
		require.NoError(t, err)

		dropDatabase(context.Background(), mSrv)

		infoSessID := insertFutureSession(t, mSrv, 1)
		insertRandSignups(t, mSrv, infoSessID, 10)

		got, err := mSrv.getUpcomingSessions(context.Background(), 1)
		require.NoError(t, err)

		require.Len(t, got[0].Participants, 10)
		for _, p := range got[0].Participants {
			require.Regexp(t, regexp.MustCompile(`\w+@\w`), p.Email, "contains a valid email address")
		}
	})
}

func insertFutureSession(t *testing.T, m *MongoService, daysInFuture int) primitive.ObjectID {
	s := Session{
		ID:        primitive.NewObjectID(),
		ProgramID: INFO_SESSION_PROGRAM_ID,
		CreatedAt: time.Now(),
	}
	s.Times.Start.DateTime = time.Now().AddDate(0, 0, daysInFuture)

	res, err := m.client.
		Database(m.dbName).Collection("sessions").
		InsertOne(context.Background(), s)
	require.NoError(t, err)

	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		require.NoError(t, errors.New("insertedID is not an ObjectID"))
	}
	return id
}

func insertRandSignups(t *testing.T, m *MongoService, infoSessionID primitive.ObjectID, n int) error {
	signups := make([]interface{}, 0)
	for i := 0; i < n; i++ {
		attendee := gofakeit.Person()
		signups = append(signups, Signup{
			CreatedAt:   time.Now(),
			ID:          primitive.NewObjectID(),
			SessionID:   infoSessionID,
			Email:       attendee.Contact.Email,
			NameFirst:   attendee.FirstName,
			NameLast:    attendee.LastName,
			FullName:    fmt.Sprintf("%s %s", attendee.FirstName, attendee.LastName),
			Cell:        gofakeit.Phone(),
			ZoomJoinURL: mustFakeZoomURL(t),
		})
	}

	_, err := m.client.Database(m.dbName).Collection("signups").
		InsertMany(context.Background(), signups)
	return err
}

func dropDatabase(ctx context.Context, m *MongoService) error {
	return m.client.Database(m.dbName).Drop(ctx)
}

func mustFakeZoomURL(t *testing.T) string {
	rand.NewSource(time.Now().Unix())
	id := rand.Intn(int(math.Pow10(11)))
	fmt.Println(id)
	return fmt.Sprintf("https://us06web.zoom.us/w/%d?tk=%s.%s", id, mustRandHex(t, 43), mustRandHex(t, 20))
}

func mustRandHex(t *testing.T, n int) string {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	require.NoError(t, err)
	return hex.EncodeToString(bytes)
}
