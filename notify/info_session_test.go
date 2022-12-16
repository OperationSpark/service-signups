package notify

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
)

type (
	MockSMSService struct {
		called     bool
		calledWith []string
	}
)

func (m *MockSMSService) Send(ctx context.Context, toNum string, msg string) error {
	m.called = true
	m.calledWith = []string{toNum, msg}
	return nil
}

func TestGetUpcomingSessions(t *testing.T) {
	t.Run("Only Retrieves sessions in the given time frame", func(t *testing.T) {
		mSrv := &MongoService{
			dbName: dbName,
			client: dbClient,
		}

		dropDatabase(context.Background(), mSrv)

		// Insert 10 sessions over the next 10 days.
		for daysInFuture := 1; daysInFuture <= 10; daysInFuture++ {
			insertFutureSession(t, mSrv, time.Hour*24*time.Duration(daysInFuture))
		}

		daysOut := 7
		inTheNextWeek := time.Hour * 24 * time.Duration(daysOut)
		sessions, err := mSrv.GetUpcomingSessions(context.Background(), inTheNextWeek)
		require.NoError(t, err)

		require.WithinRange(
			t,
			sessions[0].Times.Start.DateTime,
			time.Now(),
			time.Now().Add(inTheNextWeek),
		)

		require.Len(t, sessions, daysOut, fmt.Sprintf("should find the a session for each of the next %d days", daysOut))
	})

	t.Run("session responses contain the expected attendees of the given session", func(t *testing.T) {
		mSrv := &MongoService{
			dbName: dbName,
			client: dbClient,
		}

		dropDatabase(context.Background(), mSrv)

		infoSessID := insertFutureSession(t, mSrv, time.Hour*24)
		insertRandSignups(t, mSrv, infoSessID, 10)

		daysOut := time.Hour * 24 * 10
		got, err := mSrv.GetUpcomingSessions(context.Background(), daysOut)
		require.NoError(t, err)

		require.Len(t, got[0].Participants, 10)
		for _, p := range got[0].Participants {
			require.Regexp(t, regexp.MustCompile(`\w+@\w`), p.Email, "contains a valid email address")
		}
	})
}

func TestServer(t *testing.T) {
	t.Run("Sends SMS messages to attendees of any upcoming sessions", func(t *testing.T) {
		var body bytes.Buffer
		e := json.NewEncoder(&body)
		e.Encode(Request{
			JobName: "info-session-reminder",
			JobArgs: JobArgs{Period: "1 hour"},
		})
		req := mustMakeReq(t, &body)
		resp := httptest.NewRecorder()

		mockTwilio := MockSMSService{}
		mongoService := NewMongoService(dbClient, dbName)

		sessionID := insertFutureSession(t, mongoService, time.Hour)
		attendee := gofakeit.Person()
		toPhone := gofakeit.Phone()
		fakeSignup := Signup{
			CreatedAt:   time.Now(),
			ID:          randID(),
			SessionID:   sessionID,
			Email:       attendee.Contact.Email,
			NameFirst:   attendee.FirstName,
			NameLast:    attendee.LastName,
			FullName:    fmt.Sprintf("%s %s", attendee.FirstName, attendee.LastName),
			Cell:        toPhone,
			ZoomJoinURL: mustFakeZoomURL(t),
		}

		_, err := mongoService.client.Database(mongoService.dbName).Collection("signups").InsertOne(context.Background(), fakeSignup)
		require.NoError(t, err)

		srv := NewServer(ServerOpts{
			Store:      mongoService,
			SMSService: &mockTwilio,
		})

		srv.ServeHTTP(resp, req)

		require.Equal(t, resp.Result().StatusCode, http.StatusOK)
		require.True(t, mockTwilio.called)
		require.Contains(t, mockTwilio.calledWith, toPhone)
		// TODO:
		// require.Contains(t, mockTwilio.calledWith, "Hi from Op Spark. We're sending a friendly reminder that you signed up for an Info Session today.")
	})
}

func insertFutureSession(t *testing.T, m *MongoService, inFuture time.Duration) string {
	s := Session{
		ID:        randID(),
		ProgramID: INFO_SESSION_PROGRAM_ID,
		CreatedAt: time.Now(),
	}
	s.Times.Start.DateTime = time.Now().Add(inFuture)

	res, err := m.client.
		Database(m.dbName).Collection("sessions").
		InsertOne(context.Background(), s)
	require.NoError(t, err)

	id, ok := res.InsertedID.(string)
	if !ok {
		require.NoError(t, errors.New("insertedID is not a string"))
	}
	return id
}

func insertRandSignups(t *testing.T, m *MongoService, infoSessionID string, n int) error {
	signups := make([]interface{}, 0)
	for i := 0; i < n; i++ {
		attendee := gofakeit.Person()
		signups = append(signups, Signup{
			CreatedAt:   time.Now(),
			ID:          randID(),
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

func mustMakeReq(t *testing.T, body io.Reader) *http.Request {
	req, err := http.NewRequest(http.MethodPost, "/notify", body)
	require.NoError(t, err)
	return req
}
