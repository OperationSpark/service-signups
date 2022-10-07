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
		domain,
		fmt.Sprintf("Operation Spark <admissions@%s>", domain),
		"info-session-signup",
		mgClient,
	}
}

func (m MailgunService) run(su Signup) error {
	return m.sendWelcome(su)
}

func (m MailgunService) name() string {
	return "mailgun service"
}

func (m MailgunService) sendWelcome(su Signup) error {
	isStagingEnv := os.Getenv("APP_ENV") == "staging"

	vars, err := su.welcomeData()
	if err != nil {
		return fmt.Errorf("sendWelcome: welcomeData: %v", err)
	}

	t := mgTemplate{
		name: m.defaultTemplate,
		variables: map[string]string{
			"firstName":   vars.FirstName,
			"lastName":    vars.LastName,
			"sessionTime": vars.SessionTime,
			"sessionDate": vars.SessionDate,
			"zoomURL":     vars.ZoomURL,
		},
	}

	if isStagingEnv {
		t.version = "dev"
	}

	return m.sendWithTemplate(t, su.Email)
}

type mgTemplate struct {
	name      string            // Name of mailgun template.
	variables map[string]string // KV pairs of variables used in the email template.
	version   string            // Mailgun template version. If not set, the active version is used.
}

func (m MailgunService) sendWithTemplate(t mgTemplate, recipient string) error {
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
			return fmt.Errorf("add template variable: %v ", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Send the message with a 10 second timeout
	respMsg, id, err := m.mgClient.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("sendWithTemplate: send: %v", err)
	}

	fmt.Printf("mailgun message queued.\nID: %s Resp: %s\n\n", id, respMsg)

	return nil
}
