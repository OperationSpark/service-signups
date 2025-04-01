// Package notify provides a service for sending notifications to participants.
package notify

//lint:file-ignore SA1029 Our string context keys are unique to this package.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/operationspark/service-signup/greenlight"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/sync/errgroup"
)

type (
	Participant struct {
		NameFirst           string `bson:"nameFirst"`
		NameLast            string `bson:"nameLast"`
		FullName            string `bson:"fullName"`
		Cell                string `bson:"cell"`
		Email               string `bson:"email"`
		ZoomJoinURL         string `bson:"zoomJoinUrl"`
		SessionDate         time.Time
		SessionLocationType string
		SessionLocation     Location
	}

	UpcomingSession struct {
		ID           string           `bson:"_id"`
		ProgramID    string           `bson:"programId"`
		Times        greenlight.Times `bson:"times"`
		Participants []Participant
		LocationID   string `bson:"locationId"`
		LocationType string `bson:"locationType"`
		Location     Location
	}

	Location struct {
		Name         string `json:"name"`
		Line1        string `json:"line1"`
		CityStateZip string `json:"cityStateZip"`
		MapURL       string `json:"mapUrl"`
	}
	MongoService struct {
		dbName string
		client *mongo.Client
	}

	OSRenderer interface {
		// CreateMessageURL creates a URL to the Operation Spark Message Template Renderer with URL encoded data.
		CreateMessageURL(Participant) (string, error)
	}

	Store interface {
		GetUpcomingSessions(context.Context, time.Duration) ([]*UpcomingSession, error)
	}

	Shortener interface {
		ShortenURL(ctx context.Context, url string) (string, error)
	}

	ServerOpts struct {
		OSRendererService OSRenderer
		ShortLinkService  Shortener
		SMSService        SMSSender
		Store             Store
		Logger            *slog.Logger
	}

	SMSSender interface {
		Send(ctx context.Context, toNum, msg string) error
		FormatCell(string) string
	}

	Server struct {
		osMsSvc       OSRenderer
		shortySrv     Shortener
		store         Store
		twilioService SMSSender
		logger        *slog.Logger
	}

	Request struct {
		JobName string  `json:"jobName"`
		JobArgs JobArgs `json:"jobArgs"`
	}

	JobArgs struct {
		Period Period `json:"period"`
		DryRun bool   `json:"dryRun"`
	}

	Period string

	contextKey string
)

const (
	contextKeyRecipientTZ contextKey = "recipient_tz"
)

func (c contextKey) String() string {
	return "notify__" + string(c)
}

const (
	InfoSessionProgramID = "5sTmB97DzcqCwEZFR"
	CentralTZName        = "America/Chicago"
)

func NewServer(o ServerOpts) *Server {
	return &Server{
		osMsSvc:       o.OSRendererService,
		shortySrv:     o.ShortLinkService,
		store:         o.Store,
		twilioService: o.SMSService,
		logger:        o.Logger,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	var reqBody Request
	err := reqBody.fromJSON(r.Body)
	if err != nil {
		s.logRequestError(r.Context(), r, fmt.Errorf("request.fromJSON: %w", err))
		s.badRequestResponse(w, r, err.Error())
		return
	}

	// Remind attendees for some period in the future.
	// (1 hour, 2 days, etc)
	inFuture, err := reqBody.JobArgs.Period.Parse()
	if err != nil {
		s.logRequestError(r.Context(), r, fmt.Errorf("parse: %v, bodyL%+v", err, reqBody))
		s.badRequestResponse(w, r, err.Error())
		return
	}

	// Add timezone to the request context.
	// TODO: Base the TZ on some location information somewhere
	tz, err := time.LoadLocation(CentralTZName)
	if err != nil {
		s.serverErrorResponse(w, r, fmt.Errorf("loadLocation: %v", err))
		return
	}
	ctx := context.WithValue(r.Context(), contextKeyRecipientTZ.String(), tz)
	sessions, err := s.store.GetUpcomingSessions(ctx, inFuture)
	if err != nil {
		s.serverErrorResponse(w, r, fmt.Errorf("store.GetUpcomingSessions: %v", err))
		return
	}

	if len(sessions) == 0 {
		s.notFoundResponse(w, r, fmt.Sprintf("no upcoming sessions in the next %s", reqBody.JobArgs.Period))
		return
	}

	for _, sess := range sessions {
		s.logger.InfoContext(r.Context(), "upcoming info session",
			slog.String("sessionId", sess.ID),
			slog.String("sessionTime", sess.Times.Start.DateTime.Format(time.RubyDate)),
		)
	}

	err = s.sendSMSReminders(ctx, sessions, reqBody.JobArgs.DryRun)
	if err != nil {
		s.serverErrorResponse(w, r, fmt.Errorf("sendSMSReminders: %v", err))
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
	locations := m.client.Database(m.dbName).Collection("locations")
	for _, session := range upcomingSessions {
		suCur, err := signups.Find(ctx, bson.M{"sessionId": session.ID})
		if err != nil {
			return upcomingSessions, fmt.Errorf("signups.Find: %w", err)
		}

		// Get associated Location data
		var loc greenlight.Location
		res := locations.FindOne(ctx, bson.M{"_id": session.LocationID})
		if res.Err() != nil {
			return upcomingSessions, fmt.Errorf("locations.findOne: %w\nlocationId: %q", err, session.LocationID)
		}
		err = res.Decode(&loc)
		if err != nil {
			return upcomingSessions, fmt.Errorf("decode location: %w", err)
		}

		var attendees []Participant
		if err = suCur.All(ctx, &attendees); err != nil {
			return upcomingSessions, fmt.Errorf("signups.cursor.All(): %w", err)
		}

		for _, p := range attendees {
			p.SessionDate = session.Times.Start.DateTime
			p.SessionLocationType = session.LocationType
			p.SessionLocation = transformLocation(loc)
			session.Participants = append(session.Participants, p)
		}
	}
	return upcomingSessions, nil
}

// SendSMSReminders sends an SMS message to each of the attendees in each of the given sessions. Each SMS is sent in it's own goroutine.
func (s *Server) sendSMSReminders(ctx context.Context, sessions []*UpcomingSession, dryRun bool) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Pass in a background context to allow individual goroutines to use a new context with a timeout
	errs, _ := errgroup.WithContext(context.Background())
	for _, session := range sessions {
		for _, p := range session.Participants {
			// https://stackoverflow.com/questions/40326723/go-vet-range-variable-captured-by-func-literal-when-using-go-routine-inside-of-f
			errs.Go(func(p Participant) func() error {
				return func() error {
					// Create a new context for each goroutine
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()

					msg, err := reminderMsg(ctx, *session)
					if err != nil {
						return fmt.Errorf("reminderMsg: %w", err)
					}

					infoURL, err := s.osMsSvc.CreateMessageURL(p)
					if err != nil {
						return fmt.Errorf("osMsSvc.CreateMessageURL: %w", err)
					}

					// The link will be a long URL even if there is an error
					link, err := s.shortySrv.ShortenURL(ctx, infoURL)
					if err != nil {
						s.logError(ctx, fmt.Errorf("shortenURL %q: %w", infoURL, err))
					}

					msg = fmt.Sprintf("%s\nMore details: %s", msg, link)
					toNum := s.twilioService.FormatCell(p.Cell)
					if dryRun {
						s.logger.InfoContext(ctx, "Dry Run Mode: (not sending SMS)",
							slog.String("toNum", toNum),
							slog.String("smsBody", msg),
						)
					}
					return s.twilioService.Send(ctx, toNum, msg)
				}
			}(p))
		}
	}
	return errs.Wait()
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func reminderMsg(ctx context.Context, session UpcomingSession) (string, error) {
	tz, ok := ctx.Value(contextKeyRecipientTZ.String()).(*time.Location)
	if !ok {
		return "", errors.New("could not retrieve local timezone from context")
	}

	day := session.Times.Start.DateTime.In(tz).Format("Monday")
	time := session.Times.Start.DateTime.In(tz).Format("3:04PM MST")
	date := session.Times.Start.DateTime.In(tz).Format(" 1/2")
	if isToday(session.Times.Start.DateTime) {
		day = "today"
		date = ""
	}

	return fmt.Sprintf("Hi from Operation Spark! A friendly reminder that you have an Intro to Coding Info Session %s%s at %s.", day, date, time), nil
}

// IsToday is checks if the given time is today.
func isToday(date time.Time) bool {
	return time.Now().Before(date) && date.Before(time.Now().Add(time.Hour*13))
}

func (r *Request) fromJSON(body io.Reader) error {
	d := json.NewDecoder(body)
	return d.Decode(r)
}

// Parse parses the Period string ("3 hours") into a time.Duration.
// Acceptable periods are "day(s)", "hour(s)", "minute(s)", "min(s)".
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
	if strings.Contains(string(p), "min") {
		return time.Minute * time.Duration(val), nil
	}
	return time.Duration(0), fmt.Errorf(`invalid period type: %s\nacceptable types are "day(s)", "hour(s)", "minute(s)", "min(s)"`, parts[1])
}

func transformLocation(loc greenlight.Location) Location {
	line1, cityStateZip := greenlight.ParseAddress(loc.GooglePlace.Address)
	mapURL := greenlight.GoogleLocationLink(loc.GooglePlace.Address)
	return Location{
		Name:         loc.GooglePlace.Name,
		Line1:        line1,
		CityStateZip: cityStateZip,
		MapURL:       mapURL,
	}
}
