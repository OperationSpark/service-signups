package signup

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mailgun/mailgun-go/v4"
)

type MailgunService struct {
	domain          string               // Mail domain name.
	defaultSender   string               // Default sender email address.
	defaultTemplate string               // Default email template use when calling SendWelcome().
	mgClient        *mailgun.MailgunImpl // Mailgun API Client
}

func NewMailgunService(domain, apiKey, baseAPIurlOverride string) *MailgunService {
	mgClient := mailgun.NewMailgun(domain, apiKey)
	if len(baseAPIurlOverride) > 0 {
		mgClient.SetAPIBase(baseAPIurlOverride)
	}
	return &MailgunService{
		domain:          domain,
		defaultSender:   fmt.Sprintf("Operation Spark <admissions@%s>", domain),
		defaultTemplate: "info-session-signup",
		mgClient:        mgClient,
	}
}

func (m MailgunService) run(ctx context.Context, su Signup) error {
	return m.sendWelcome(ctx, su)
}

func (m MailgunService) name() string {
	return "mailgun service"
}

func (m MailgunService) sendWelcome(ctx context.Context, su Signup) error {
	isStagingEnv := os.Getenv("APP_ENV") == "staging"

	vars, err := su.welcomeData()
	if err != nil {
		return fmt.Errorf("welcomeData: %w", err)
	}

	t := mgTemplate{
		name: m.defaultTemplate,
		variables: map[string]string{
			"firstName":            vars.FirstName,
			"lastName":             vars.LastName,
			"sessionTime":          vars.SessionTime,
			"sessionDate":          vars.SessionDate,
			"zoomURL":              vars.ZoomURL,
			"locationLine1":        vars.LocationLine1,
			"locationCityStateZip": vars.LocationCityStateZip,
			"locationMapUrl":       vars.LocationMapURL,
		},
	}
	// add joinCode to variables if it exists
	if vars.JoinCode != "" {
		t.variables["joinCode"] = vars.JoinCode
	}

	if su.LocationType == "HYBRID" {
		t.name = "info-session-signup-hybrid"
	}

	if isStagingEnv {
		t.version = "dev"
	}

	return m.sendWithTemplate(ctx, t, su.Email)
}

type mgTemplate struct {
	name      string            // Name of mailgun template.
	variables map[string]string // KV pairs of variables used in the email template.
	version   string            // Mailgun template version. If not set, the active version is used.
}

func (m MailgunService) sendWithTemplate(ctx context.Context, t mgTemplate, recipient string) error {
	sender := m.defaultSender
	subject := "Welcome from Operation Spark!"
	// Empty body because we're using a template
	body := ""

	message := m.mgClient.NewMessage(sender, subject, body, recipient)
	message.SetTemplate(t.name)
	if len(t.version) > 0 {
		message.SetTemplateVersion(t.version)
	}
	for k, v := range t.variables {
		err := message.AddTemplateVariable(k, v)
		if err != nil {
			return fmt.Errorf("add template variable: %w ", err)
		}
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	// Send the message with a 10 second timeout
	_, _, err := m.mgClient.Send(ctxWithTimeout, message)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	return nil
}
