package notify

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
		uri    string
	}

	SMSSender interface {
		Send(ctx context.Context, toNum, msg string) error
	}

	Server struct {
		twilioService SMSSender
		mongoService  *MongoService
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

	ServerOpts struct {
		MongoURI      string
		TwilioService SMSSender
	}
)

const INFO_SESSION_PROGRAM_ID = "5sTmB97DzcqCwEZFR"

func NewServer(o ServerOpts) *Server {
	mongoSvc, err := NewMongoService(context.Background(), o.MongoURI)
	if err != nil && os.Getenv("CI") != "true" {
		log.Fatalf("Could not connect to MongoDB: %s", o.MongoURI)
	}
	return &Server{
		mongoService:  mongoSvc,
		twilioService: o.TwilioService,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("->%s %s\n", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	// Remind attendees for today's info session(s)
	// Get from request body/params?
	hoursInFuture := time.Hour * 24 * 1
	sessions, err := s.mongoService.getUpcomingSessions(r.Context(), hoursInFuture)
	if err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for _, sess := range sessions {
		fmt.Printf("upcoming info session:\nID: %s, Time: %s\n",
			sess.ID,
			sess.Times.Start.DateTime.Format(time.RubyDate))
	}

	err = s.sendSMSReminders(r.Context(), sessions)
	if err != nil {
		fmt.Println("One or more message failed to send", err)
		http.Error(w, "One or more message failed to send", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func NewMongoService(ctx context.Context, uri string) (*MongoService, error) {
	parsed, err := url.Parse(uri)
	isCI := os.Getenv("CI") == "true"
	if (!isCI && uri == "") || err != nil {
		log.Fatalf("Invalid 'MONGO_URI' environmental variable: %q", uri)
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	return &MongoService{
		dbName: strings.TrimPrefix(parsed.Path, "/"),
		uri:    uri,
		client: client,
	}, nil
}

// GetUpcomingSessions queries the database for Info Sessions starting between not and some time in the future. Returns the upcoming Info Sessions and the email addresses of each session's prospective participants.
func (m *MongoService) getUpcomingSessions(ctx context.Context, inFuture time.Duration) ([]*UpcomingSession, error) {
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
					msg := "go to the info session"
					return s.twilioService.Send(ctx, p.Cell, msg)
				}
			}(p))
		}
	}
	return errs.Wait()
}
