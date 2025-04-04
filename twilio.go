package signup

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/twilio/twilio-go"
	"github.com/twilio/twilio-go/client"
	conversations "github.com/twilio/twilio-go/rest/conversations/v1"
)

type (
	smsService struct {
		// Twilio API base URL. This is used as an override for testing API calls.
		apiBase string
		// Client for making requests to Twilio's API.
		client *twilio.RestClient
		// Phone number SMS messages are sent from.
		fromPhoneNum string
		// API base for Operation Spark's SMS Messaging interface.
		// This URL is used for sending webhooks on SMS events from this service.
		// Default: https://messenger.operationspark.org
		opSparkMessagingSvcBaseURL string
		// Twilio Conversation (Chat) Service ID.
		// Ex: "IS00000000000000000000000000000000"
		conversationsSid string
		// Twilio Conversations Service User identity name.
		// Ex: "services@operationspark.org"
		conversationsIdentity string
	}

	// ErrInvalidNumber is an error type for invalid phone numbers.
	ErrInvalidNumber struct {
		err error
	}

	twilioServiceOptions struct {
		accountSID string
		authToken  string
		// Client for making requests to Twilio's API.
		client client.BaseClient
		// Phone number SMS messages are sent from.
		fromPhoneNum string
		// API base for Operation Spark's SMS Messaging interface.
		// This URL is used for sending webhooks on SMS events from this service.
		// Default: https://messenger.operationspark.org
		opSparkMessagingSvcBaseURL string
		// Twilio API base.
		apiBase          string
		conversationsSid string
		// Twilio Conversations Service User identity name.
		// Ex: "services@operationspark.org"
		conversationsIdentity string
	}
)

func (e ErrInvalidNumber) Error() string {
	return e.err.Error()
}

func NewTwilioService(o twilioServiceOptions) *smsService {
	messengerBaseURL := "https://messenger.operationspark.org"
	if len(o.opSparkMessagingSvcBaseURL) > 0 {
		messengerBaseURL = o.opSparkMessagingSvcBaseURL
	}

	// Override for testing
	apiBase := "https://api.twilio.com"
	if len(o.apiBase) > 0 {
		apiBase = o.apiBase
	}

	conversationsIdentity := "services@operationspark.org"
	if len(o.conversationsIdentity) > 0 {
		conversationsIdentity = o.conversationsIdentity
	}

	return &smsService{
		apiBase: apiBase,
		client: twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: o.accountSID,
			Password: o.authToken,
			Client:   o.client,
		}),
		fromPhoneNum:               o.fromPhoneNum,
		opSparkMessagingSvcBaseURL: messengerBaseURL,
		conversationsSid:           o.conversationsSid,
		conversationsIdentity:      conversationsIdentity,
	}
}

// IsRequired returns false because we're now going to send the information URL back to the client in the response body. So if the SMS message fails to send, the user will still have the information URL.
func (t smsService) isRequired() bool {
	return false
}

// Run sends an Info Session signup confirmation SMS to the registered participant. We use Twilio's Conversations API instead of the Messaging API to allow multiple staff members communicate with the participant through the same outgoing SMS number.
// The confirmation SMS contains a link that when clicked generates a custom page containing information on the upcoming Info Session. This signup-specific link is shortened before sent.
//
// Note: Twilio has a free Link Shortening service, but it is only available with the Messaging API, not Conversations.
func (t *smsService) run(ctx context.Context, su *Signup, logger *slog.Logger) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if !su.SMSOptIn {
		logger.InfoContext(ctx, "User opted-out from SMS messages")
		return nil
	}

	toNum := t.FormatCell(su.Cell)
	convoName := fmt.Sprintf("%s %s", su.NameFirst, su.NameLast[0:1])
	convoID := ""
	existing, err := t.findConversationsByNumber(toNum)
	if err != nil {
		return fmt.Errorf("findConversationsByNumber: %w", err)
	}

	// Create a new conversation if none exists
	if len(existing) == 0 {
		convoID, err = t.addNumberToConversation(toNum, convoName)
		if err != nil {
			twilioInvalidPhoneCode := "50407"
			// check if error is due to number being invalid, if so use the errInvalidNumber type
			if strings.Contains(err.Error(), twilioInvalidPhoneCode) {
				return ErrInvalidNumber{err: fmt.Errorf("invalid number: %s", toNum)}
			}

			return fmt.Errorf("addNumberToConversation: %w", err)
		}
	} else {
		// TODO: Fix this potentially faulty logic if picking the first existing conversation
		convoID = *existing[0].ConversationSid
	}

	// Send Opt-in confirmation
	if err := t.optInConfirmation(ctx, toNum); err != nil {
		return fmt.Errorf("optInConfirmation: %w", err)
	}

	if su.ShortLink == "" {
		// This should never happen
		return fmt.Errorf("shortLink is empty")
	}
	// Create the SMS message body
	msg, err := su.shortMessage(su.ShortLink)
	if err != nil {
		return fmt.Errorf("shortMessage: %w", err)
	}

	err = t.sendSMSInConversation(msg, convoID)
	if err != nil {
		return fmt.Errorf("sendSMS: %w", err)
	}

	err = t.sendConvoWebhook(ctx, convoID)
	if err != nil {
		logger.ErrorContext(ctx, fmt.Errorf("sendConvoWebhook (messenger API): %w", err).Error())
	}
	// Carry on even if the Messenger API webhook fails

	// Set the conversation ID on the signup record
	su.conversationID = &convoID

	return nil
}

func (t *smsService) name() string {
	return "twilio service"
}

// SendSMSInConversation uses the Twilio Conversations API to send a message to a specific Conversation. Twilio will then broadcast the message to the Conversation participants. In our case, this is two SMS-capable phone numbers.
func (t *smsService) sendSMSInConversation(body string, convoID string) error {
	params := &conversations.CreateServiceConversationMessageParams{
		Body:   &body,
		Author: &t.conversationsIdentity,
	}

	_, err := t.client.ConversationsV1.CreateServiceConversationMessage(t.conversationsSid, convoID, params)
	if err != nil {
		return fmt.Errorf("createServiceConversationMessage: %w", err)
	}

	return nil
}

// FindConversationsByNumber finds all Twilio Conversations that have the given phone number as a participant.
func (t *smsService) findConversationsByNumber(phNum string) ([]conversations.ConversationsV1ServiceParticipantConversation, error) {
	params := &conversations.ListServiceParticipantConversationParams{}
	params.SetAddress(phNum)
	params.SetLimit(20)

	resp, err := t.client.ConversationsV1.ListServiceParticipantConversation(t.conversationsSid, params)
	if err != nil {
		return resp, fmt.Errorf("listServiceParticipantConversation: %w", err)
	}
	return resp, nil
}

// AddNumberToConversation creates a new Conversation and adds two participants - the Operation Spark Service Identity ("services@operationspark.org"), and the SMS recipient's phone number.
func (t *smsService) addNumberToConversation(phNum, friendlyName string) (string, error) {
	cp := &conversations.CreateServiceConversationParams{}
	cp.SetFriendlyName(friendlyName)

	// Create new Conversation
	cResp, err := t.client.ConversationsV1.CreateServiceConversation(t.conversationsSid, cp)
	if err != nil {
		return "", fmt.Errorf("createServiceConversation: %w", err)
	}

	// Add Operation Spark Conversation Identity
	ppp := &conversations.CreateServiceConversationParticipantParams{}
	ppp.SetIdentity(t.conversationsIdentity)
	_, err = t.client.ConversationsV1.CreateServiceConversationParticipant(t.conversationsSid, *cResp.Sid, ppp)
	if err != nil {
		return "", fmt.Errorf("createServiceConversationParticipant with Service Identity: %w: ", err)
	}

	// Add SMS Recipient to conversation
	pp := &conversations.CreateServiceConversationParticipantParams{}
	pp.SetMessagingBindingAddress(phNum)
	pp.SetMessagingBindingProxyAddress(t.fromPhoneNum)
	friendlyNameWithNum := fmt.Sprintf("%s (%s)", friendlyName, phNum)
	pp.SetAttributes(fmt.Sprintf(`{"friendlyName": %q}`, friendlyNameWithNum))

	_, err = t.client.ConversationsV1.CreateServiceConversationParticipant(t.conversationsSid, *cResp.Sid, pp)
	if err != nil {
		return "", fmt.Errorf("createServiceConversationParticipant: %w\nidentity: %q", err, phNum)
	}

	return *cResp.Sid, nil
}

// FormatCell prepends the US country code, "+1", and removes any dashes from a phone number string.
func (t *smsService) FormatCell(cell string) string {
	return "+1" + strings.ReplaceAll(cell, "-", "")
}

// SendConvoWebhook sends a webhook to OS Messaging Service to indicate a new Conversation was created.
func (t *smsService) sendConvoWebhook(ctx context.Context, convoID string) error {
	url := fmt.Sprintf("%s/api/webhooks/conversation/%s", t.opSparkMessagingSvcBaseURL, convoID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("newRequest: %w", err)
	}
	req.Header.Add("key", os.Getenv("URL_SHORTENER_API_KEY"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do: %w", err)
	}

	if resp.StatusCode >= 300 {
		return handleHTTPError(resp)
	}
	return nil
}

func (t *smsService) optInConfirmation(ctx context.Context, toNum string) error {
	msg := "You've opted in for texts from Operation Spark for upcoming sessions. You can text us here if you have further questions. Message and data rates may apply. Reply STOP to unsubscribe."
	return t.Send(ctx, toNum, msg)
}

// Send sends an SMS message to the given toNum and returns an error.
func (t *smsService) Send(ctx context.Context, toNum string, msg string) error {
	// TODO: Maybe consolidate this code with some of the run() code
	convoID := ""
	existing, err := t.findConversationsByNumber(toNum)
	if err != nil {
		return fmt.Errorf("findConversationsByNumber: %w", err)
	}

	// Create a new conversation if none exists
	// I think we should always already have an existing conversation
	if len(existing) == 0 {
		convoID, err = t.addNumberToConversation(toNum, toNum)
		if err != nil {
			return fmt.Errorf("addNumberToConversation: %w", err)
		}
	} else {
		if len(existing) > 1 {
			return fmt.Errorf("found more than one existing conversation for cell: %q. %+v", toNum, existing)
		}
		// There can only be one..
		convoID = *existing[0].ConversationSid
	}
	err = t.sendSMSInConversation(msg, convoID)
	if err != nil {
		return fmt.Errorf("sendSMS: %w", err)
	}
	return nil
}
