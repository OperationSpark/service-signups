package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/sync/errgroup"
)

type (
	Participant struct {
		NameFirst   string `bson:"nameFirst"`
		NameLast    string `bson:"nameLast"`
		FullName    string `bson:"fullName"`
		Cell        string `bson:"cell"`
		Email       string `bson:"email"`
		ZoomJoinURL string `bson:"zoomJoinUrl"`
	}

	UpcomingSession struct {
		ID           string `bson:"_id"`
		ProgramID    string `bson:"programId"`
		Times        Times  `bson:"times"`
		Participants []Participant
	}

	MongoService struct {
		dbName string
		client *mongo.Client
	}

	SMSSender interface {
		Send(ctx context.Context, toNum, msg string) error
		FormatCell(string) string
	}

	Server struct {
		twilioService SMSSender
		store         Store
	}

	Times struct {
		Start struct {
			DateTime time.Time `bson:"dateTime"`
		} `bson:"start"`
	}

	Session struct {
		ID        string    `bson:"_id"`
		ProgramID string    `bson:"programId"`
		Times     Times     `bson:"times"` // TODO: Check out "inline" struct tag
		Cohort    string    `bson:"cohort"`
		Students  []string  `bson:"students"`
		Name      string    `bson:"name"`
		CreatedAt time.Time `bson:"createdAt"`
	}

	Signup struct {
		// Legacy Meteor did not use Mongo's ObjectID() _id creation.
		ID          string    `bson:"_id"`
		SessionID   string    `bson:"sessionId"`
		NameFirst   string    `bson:"nameFirst"`
		NameLast    string    `bson:"nameLast"`
		FullName    string    `bson:"fullName"`
		Cell        string    `bson:"cell"`
		Email       string    `bson:"email"`
		CreatedAt   time.Time `bson:"createdAt"`
		ZoomJoinURL string    `bson:"zoomJoinUrl"`
	}

	Store interface {
		GetUpcomingSessions(context.Context, time.Duration) ([]*UpcomingSession, error)
	}

	ServerOpts struct {
		Store      Store
		SMSService SMSSender
	}

	Request struct {
		JobName string  `json:"jobName"`
		JobArgs JobArgs `json:"jobArgs"`
	}

	JobArgs struct {
		Period Period `json:"period"`
	}

	Period string

	contextKey int
)

const (
	RECIPIENT_TZ contextKey = iota
)

const (
	INFO_SESSION_PROGRAM_ID = "5sTmB97DzcqCwEZFR"
	CENTRAL_TZ_NAME         = "America/Chicago"
)

func NewServer(o ServerOpts) *Server {
	return &Server{
		store:         o.Store,
		twilioService: o.SMSService,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	// Add timezone to the request context.
	// TODO: Base the TZ on some location information somewhere
	tz, err := time.LoadLocation(CENTRAL_TZ_NAME)
	if err != nil {
		fmt.Printf("loadLocation: %v, ", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	ctx := context.WithValue(r.Context(), RECIPIENT_TZ, tz)
	var reqBody Request
	reqBody.fromJSON(r.Body)

	// Remind attendees for some period in the future.
	// (1 hour, 2 days, etc)
	inFuture, err := reqBody.JobArgs.Period.Parse()
	if err != nil {
		fmt.Printf("parse: %v, bodyL%+v\n", err, reqBody)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sessions, err := s.store.GetUpcomingSessions(ctx, inFuture)
	if err != nil {
		fmt.Printf("getUpcomingSessions: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for _, sess := range sessions {
		fmt.Printf("upcoming info session:\nID: %s, Time: %s\n",
			sess.ID,
			sess.Times.Start.DateTime.Format(time.RubyDate))
	}

	err = s.sendSMSReminders(ctx, sessions)
	if err != nil {
		fmt.Println("One or more message failed to send", err)
		http.Error(w, "One or more message failed to send", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func NewMongoService(dbClient *mongo.Client, dbName string) *MongoService {
	return &MongoService{
		dbName: dbName,
		client: dbClient,
	}
}

// GetUpcomingSessions queries the database for Info Sessions starting between not and some time in the future. Returns the upcoming Info Sessions and the email addresses of each session's prospective participants.
func (m *MongoService) GetUpcomingSessions(ctx context.Context, inFuture time.Duration) ([]*UpcomingSession, error) {
	sessions := m.client.Database(m.dbName).Collection("sessions")

	infoSessionProgID := "5sTmB97DzcqCwEZFR"
	filterDate := time.Now().Add(inFuture)

	var upcomingSessions []*UpcomingSession
	sessCursor, err := sessions.Find(ctx, bson.M{
		"programId": infoSessionProgID,
		"times.start.dateTime": bson.M{
			"$gte": time.Now(),
			"$lte": filterDate,
		},
	})

	if err != nil {
		return upcomingSessions, fmt.Errorf("sessions.Find: %w", err)
	}

	if err = sessCursor.All(ctx, &upcomingSessions); err != nil {
		return upcomingSessions, fmt.Errorf("sessions cursor.All(): %w", err)
	}

	// Fetch the attendees
	signups := m.client.Database(m.dbName).Collection("signups")
	for _, session := range upcomingSessions {
		cur, err := signups.Find(ctx, bson.M{"sessionId": session.ID})
		if err != nil {
			return upcomingSessions, fmt.Errorf("signups.Find: %w", err)
		}
		var attendees []Participant
		if err = cur.All(ctx, &attendees); err != nil {
			return upcomingSessions, fmt.Errorf("signups.cursor.All(): %w", err)
		}
		session.Participants = append(session.Participants, attendees...)
	}
	return upcomingSessions, nil
}

func (s *Server) sendSMSReminders(ctx context.Context, sessions []*UpcomingSession) error {
	errs, ctx := errgroup.WithContext(ctx)
	for _, session := range sessions {
		for _, p := range session.Participants {
			// https://stackoverflow.com/questions/40326723/go-vet-range-variable-captured-by-func-literal-when-using-go-routine-inside-of-f
			errs.Go(func(p Participant) func() error {
				return func() error {
					msg, err := reminderMsg(ctx, *session)
					if err != nil {
						return fmt.Errorf("reminderMsg: %w", err)
					}
					toNum := s.twilioService.FormatCell(p.Cell)
					return s.twilioService.Send(ctx, toNum, msg)
				}
			}(p))
		}
	}
	return errs.Wait()
}

func reminderMsg(ctx context.Context, session UpcomingSession) (string, error) {
	tz, ok := ctx.Value(RECIPIENT_TZ).(*time.Location)
	if !ok {
		return "", errors.New("could not retrieve local timezone from context")
	}

	day := session.Times.Start.DateTime.In(tz).Format("Monday")
	if isToday(session.Times.Start.DateTime) {
		day = "today"
	}
	time := session.Times.Start.DateTime.In(tz).Format("03:04PM MST")
	// TODO: Include short link
	return fmt.Sprintf("Hi from Operation Spark! A friendly reminder that you have an Info Session %s at %s", day, time), nil
}

func isToday(date time.Time) bool {
	return time.Now().Before(date) && date.Before(time.Now().Add(time.Hour*13))
}

func (r *Request) fromJSON(body io.Reader) error {
	d := json.NewDecoder(body)
	return d.Decode(r)
}

func (p Period) Parse() (time.Duration, error) {
	parts := strings.Fields(string(p))
	rawVal := parts[0]
	val, err := strconv.Atoi(rawVal)
	if err != nil {
		return time.Duration(0), err
	}
	if strings.Contains(string(p), "day") {
		return time.Hour * 24 * time.Duration(val), nil
	}
	if strings.Contains(string(p), "hour") {
		return time.Hour * time.Duration(val), nil
	}
	if strings.Contains(string(p), "minute") {
		return time.Minute * time.Duration(val), nil
	}
	return time.Duration(0), fmt.Errorf(`invalid period type: %s\nacceptable types are "day(s)", "hour(s)", "minute(s)"`, parts[1])
}
