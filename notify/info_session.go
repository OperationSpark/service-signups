package notify

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
		Send() error
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
		ID        primitive.ObjectID `bson:"_id"`
		ProgramID string             `bson:"programId"`
		Times     Times              `bson:"times"` // TODO: Check out "inline" struct tag
		Cohort    string             `bson:"cohort"`
		Students  []string           `bson:"students"`
		Name      string             `bson:"name"`
		CreatedAt time.Time          `bson:"createdAt"`
	}

	Signup struct {
		ID          primitive.ObjectID `bson:"_id"`
		SessionID   primitive.ObjectID `bson:"sessionId"`
		NameFirst   string             `bson:"nameFirst"`
		NameLast    string             `bson:"nameLast"`
		FullName    string             `bson:"fullName"`
		Cell        string             `bson:"cell"`
		Email       string             `bson:"email"`
		CreatedAt   time.Time          `bson:"createdAt"`
		ZoomJoinURL string             `bson:"zoomJoinUrl"`
	}

	ServerOpts struct {
		MongoURI string
	}
)

const INFO_SESSION_PROGRAM_ID = "5sTmB97DzcqCwEZFR"

func NewServer(o ServerOpts) *Server {
	mongoSvc, err := NewMongoService(context.Background(), o.MongoURI)
	if err != nil {
		log.Fatalf("Could not connect to MongoDB: %s", o.MongoURI)
	}
	return &Server{
		mongoService: mongoSvc,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("->%s %s\n", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	// Remind attendees for today's info session(s)
	sessions, err := s.mongoService.getUpcomingSessions(r.Context(), 1)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// TODO: Send the reminders
	fmt.Println("upcoming info sessions:", sessions)

	w.WriteHeader(http.StatusOK)
}

func NewMongoService(ctx context.Context, uri string) (*MongoService, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		log.Fatal("Invalid 'MONGODB_URI' environmental variable.")
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

// GetUpcomingSessions queries the database for Info Sessions starting in the next n days. Returns the upcoming Info Sessions and the email addresses of each session's prospective participants.
func (m MongoService) getUpcomingSessions(ctx context.Context, inDays int) ([]*UpcomingSession, error) {
	sessions := m.client.Database(m.dbName).Collection("sessions")

	infoSessionProgID := "5sTmB97DzcqCwEZFR"
	filterDate := time.Now().AddDate(0, 0, inDays)

	var upcomingSessions []*UpcomingSession
	sessCursor, err := sessions.Find(ctx, bson.M{
		"programId": infoSessionProgID,
		"times.start.dateTime": bson.M{
			"$gte": time.Now(),
			"$lte": filterDate,
		},
	})

	if err != nil {
		return upcomingSessions, err
	}

	if err = sessCursor.All(ctx, &upcomingSessions); err != nil {
		return upcomingSessions, err
	}

	// Fetch the attendees
	signups := m.client.Database(m.dbName).Collection("signups")
	for _, session := range upcomingSessions {
		sessionID, err := primitive.ObjectIDFromHex(session.ID)
		if err != nil {
			return upcomingSessions, err
		}
		cur, err := signups.Find(ctx, bson.M{"sessionId": sessionID})
		if err != nil {
			return upcomingSessions, err
		}
		var attendees []Participant
		if err = cur.All(ctx, &attendees); err != nil {
			return upcomingSessions, err
		}
		session.Participants = append(session.Participants, attendees...)
	}
	return upcomingSessions, nil
}
