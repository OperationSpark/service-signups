package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/operationspark/service-signup/greenlight"
	"github.com/stretchr/testify/require"
)

type (
	MockSMSService struct {
		called     bool
		calledWith []string
	}

	MockOSMessenger struct{}
	MockShortLinker struct{}
)

func (m *MockSMSService) Send(ctx context.Context, toNum string, msg string) error {
	m.called = true
	m.calledWith = []string{toNum, msg}
	return nil
}

func (m *MockSMSService) FormatCell(cell string) string {
	return "+1" + strings.ReplaceAll(cell, "-", "")
}

func (m MockOSMessenger) CreateMessageURL(Participant) (string, error) {
	return "", nil
}

func (m MockShortLinker) ShortenURL(ctx context.Context, url string) (string, error) {
	return "", nil
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
			require.Equal(t, "HYBRID", p.SessionLocationType)
			require.Equal(t, "Operation Spark", p.SessionLocation.Name)
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

		mongoService := NewMongoService(dbClient, dbName)

		sessionID := insertFutureSession(t, mongoService, time.Hour)
		attendee := gofakeit.Person()
		toPhone := attendee.Contact.Phone
		fakeSignup := greenlight.Signup{
			CreatedAt:   time.Now(),
			ID:          randID(),
			SessionID:   sessionID,
			Email:       attendee.Contact.Email,
			NameFirst:   attendee.FirstName,
			NameLast:    attendee.LastName,
			FullName:    fmt.Sprintf("%s %s", attendee.FirstName, attendee.LastName),
			Cell:        toPhone,
			ZoomJoinURL: MustFakeZoomURL(t),
		}

		_, err := mongoService.client.Database(mongoService.dbName).Collection("signups").InsertOne(context.Background(), fakeSignup)
		require.NoError(t, err)

		mockTwilio := MockSMSService{}

		srv := NewServer(ServerOpts{
			OSMessagingService: MockOSMessenger{},
			ShortLinkService:   MockShortLinker{},
			Store:              mongoService,
			SMSService:         &mockTwilio,
		})

		srv.ServeHTTP(resp, req)

		require.Equal(t, resp.Result().StatusCode, http.StatusOK)
		require.True(t, mockTwilio.called)
		require.Contains(t, mockTwilio.calledWith, mockTwilio.FormatCell(toPhone))
		require.Contains(t, mockTwilio.calledWith[1], "Hi from Operation Spark! A friendly reminder that you have an Info Session")
	})
}

func TestReminderMsg(t *testing.T) {
	t.Run(`Reminder message includes "today" if the session is today`, func(t *testing.T) {
		ctx := context.WithValue(context.Background(), RECIPIENT_TZ, time.UTC)
		session := UpcomingSession{}
		session.Times.Start.DateTime = time.Now().Add(time.Hour * 5)
		got, err := reminderMsg(ctx, session)
		require.NoError(t, err)
		want := "Hi from Operation Spark! A friendly reminder that you have an Info Session today"
		require.Contains(t, got, want)
	})

	t.Run("Reminder message includes the day of the week if the session is not today", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), RECIPIENT_TZ, time.UTC)
		session := UpcomingSession{}
		mardiGras, err := time.Parse("Jan 02, 2006", "Feb 21, 2023") // Mardi Gras
		require.NoError(t, err)

		session.Times.Start.DateTime = mardiGras

		got, err := reminderMsg(ctx, session)
		require.NoError(t, err)

		want := "Tuesday"
		require.Contains(t, got, want)

	})
}

// ** Test Helpers ** //
func insertFutureSession(t *testing.T, m *MongoService, inFuture time.Duration) string {
	location := greenlight.Location{
		ID: randID(),
	}
	location.GooglePlace = greenlight.GooglePlace{
		PlaceID: "ChIJ7YchCHSmIIYRYsAEPZN_E0o",
		Name:    "Operation Spark",
		Address: "514 Franklin Ave, New Orleans, LA 70117, USA",
		Phone:   "+1 504-534-8277",
		Website: "https://www.operationspark.org/",
		Geometry: greenlight.Geometry{
			Lat: 29.96325999999999,
			Lng: -90.052138,
		},
	}

	locRes, err := m.client.
		Database(m.dbName).Collection("locations").
		InsertOne(context.Background(), location)
	require.NoError(t, err)

	locationID, ok := locRes.InsertedID.(string)
	require.True(t, ok)
	s := greenlight.Session{
		ID:           randID(),
		ProgramID:    INFO_SESSION_PROGRAM_ID,
		CreatedAt:    time.Now(),
		LocationType: "HYBRID",
		LocationID:   locationID,
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
		signups = append(signups, greenlight.Signup{
			CreatedAt:   time.Now(),
			ID:          randID(),
			SessionID:   infoSessionID,
			Email:       attendee.Contact.Email,
			NameFirst:   attendee.FirstName,
			NameLast:    attendee.LastName,
			FullName:    fmt.Sprintf("%s %s", attendee.FirstName, attendee.LastName),
			Cell:        gofakeit.Phone(),
			ZoomJoinURL: MustFakeZoomURL(t),
		})
	}

	_, err := m.client.Database(m.dbName).Collection("signups").
		InsertMany(context.Background(), signups)
	return err
}

func dropDatabase(ctx context.Context, m *MongoService) error {
	return m.client.Database(m.dbName).Drop(ctx)
}
