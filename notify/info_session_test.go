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
	"go.mongodb.org/mongo-driver/bson"
)

type (
	MockSMSService struct {
		called     bool
		calledWith []string
	}

	MockOSRenderer  struct{}
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

func (m MockOSRenderer) CreateMessageURL(Participant) (string, error) {
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

		err := dropDatabase(context.Background(), mSrv)
		require.NoError(t, err)

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

		err := dropDatabase(context.Background(), mSrv)
		require.NoError(t, err)

		infoSessID := insertFutureSession(t, mSrv, time.Hour*24)
		err = insertRandSignups(t, mSrv, infoSessID, 10)
		require.NoError(t, err)

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

	t.Run("handles session locations with string values for 'googlePlace' field. (Schemaless legacy data)", func(t *testing.T) {
		mSrv := &MongoService{
			dbName: dbName,
			client: dbClient,
		}

		err := dropDatabase(context.Background(), mSrv)
		require.NoError(t, err)

		// Some "location.googlePlace" fields are empty strings in the database
		badLocationJSON := fmt.Sprintf(`{
			"_id":%q,
			"type": "LEARNING_CENTER",
			"googlePlace": "",
			"name": "Operation Spark-Virtual",
			"address": "",
			"city": "New Orleans",
			"state": "LA",
			"zip": "70119",
			"contact": "Admissions"
			}`, mustRandID(t))

		// JSON Doc to BSON doc
		d := json.NewDecoder(strings.NewReader(badLocationJSON))
		var locationDoc bson.M
		err = d.Decode(&locationDoc)
		require.NoError(t, err)

		locRes, err := mSrv.client.
			Database(mSrv.dbName).Collection("locations").
			InsertOne(context.Background(), locationDoc)
		require.NoError(t, err)

		locationID, ok := locRes.InsertedID.(string)
		require.True(t, ok)

		// Insert Session with associated with bad Location
		s := greenlight.Session{
			ID:           mustRandID(t),
			ProgramID:    INFO_SESSION_PROGRAM_ID,
			LocationType: "VIRTUAL",
			LocationID:   locationID,
		}
		s.Times.Start.DateTime = time.Now().Add(time.Hour)

		// Insert Session into DB
		sessRes, err := mSrv.client.
			Database(mSrv.dbName).Collection("sessions").
			InsertOne(context.Background(), s)
		require.NoError(t, err)
		sessionID, ok := sessRes.InsertedID.(string)
		require.True(t, ok, "sessionID should be a string")
		require.NotEmpty(t, sessionID)

		// Insert Signup into DB
		suRes, err := mSrv.client.
			Database(mSrv.dbName).Collection("signups").
			InsertOne(context.Background(), greenlight.Signup{
				NameFirst: "Halle",
				NameLast:  "Bot",
				SessionID: sessionID,
			})
		require.NoError(t, err)
		require.NotEmpty(t, suRes)

		require.NoError(t, err)
		require.NotEmpty(t, sessRes.InsertedID)

		// ** End of DB Setup ** //

		// ** Behavior under test ** //
		gotSessions, err := mSrv.GetUpcomingSessions(context.Background(), time.Hour*2)
		require.NoError(t, err)
		require.Len(t, gotSessions, 1, "should be 1 upcoming session")
		require.Len(t, gotSessions[0].Participants, 1, "info session should have one registered participant")
		participant := gotSessions[0].Participants[0]
		require.Equal(t, "VIRTUAL", participant.SessionLocationType)
	})
}

func TestServer(t *testing.T) {
	t.Run("Sends SMS messages to attendees of any upcoming sessions", func(t *testing.T) {
		var body bytes.Buffer
		e := json.NewEncoder(&body)
		err := e.Encode(Request{
			JobName: "info-session-reminder",
			JobArgs: JobArgs{Period: "1 hour"},
		})
		require.NoError(t, err)

		req := mustMakeReq(t, &body)
		resp := httptest.NewRecorder()

		mongoService := NewMongoService(dbClient, dbName)

		err = dropDatabase(context.Background(), mongoService)
		require.NoError(t, err)

		sessionID := insertFutureSession(t, mongoService, time.Hour)
		attendee := gofakeit.Person()
		toPhone := attendee.Contact.Phone
		fakeSignup := greenlight.Signup{
			CreatedAt:   time.Now(),
			ID:          mustRandID(t),
			SessionID:   sessionID,
			Email:       attendee.Contact.Email,
			NameFirst:   attendee.FirstName,
			NameLast:    attendee.LastName,
			FullName:    fmt.Sprintf("%s %s", attendee.FirstName, attendee.LastName),
			Cell:        toPhone,
			ZoomJoinURL: MustFakeZoomURL(t),
		}

		_, err = mongoService.client.Database(mongoService.dbName).Collection("signups").InsertOne(context.Background(), fakeSignup)
		require.NoError(t, err)

		mockTwilio := MockSMSService{}

		srv := NewServer(ServerOpts{
			OSRendererService: MockOSRenderer{},
			ShortLinkService:  MockShortLinker{},
			Store:             mongoService,
			SMSService:        &mockTwilio,
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
		want := "Hi from Operation Spark! A friendly reminder that you have an Info Session today at "
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

		want := "Tuesday 2/21 at "
		require.Contains(t, got, want)

	})
}

// ** Test Helpers ** //
func insertFutureSession(t *testing.T, m *MongoService, inFuture time.Duration) string {
	location := greenlight.Location{
		ID: mustRandID(t),
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
		ID:           mustRandID(t),
		ProgramID:    INFO_SESSION_PROGRAM_ID,
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
			ID:          mustRandID(t),
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
