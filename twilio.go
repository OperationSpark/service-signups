package signup

import (
	"context"
	"fmt"
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

	// error type for invalid phone numbers
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

// IsRequired returns true because the SMS message is required to be sent to the student so that they can attend the info session.
func (s smsService) isRequired() bool {
	return true
}

// Run sends an Info Session signup confirmation SMS to the registered participant. We use Twilio's Conversations API instead of the Messaging API to allow multiple staff members communicate with the participant through the same outgoing SMS number.
// The confirmation SMS contains a link that when clicked generates a custom page containing information on the upcoming Info Session. This signup-specific link is shortened before sent.
//
// Note: Twilio has a free Link Shortening service, but it is only available with the Messaging API, not Conversations.
func (t *smsService) run(ctx context.Context, su Signup) error {
	if !su.SMSOptIn {
		fmt.Printf("User opted-out from SMS messages: %s\n", su.String())
		return nil
	}

	toNum := t.FormatCell(su.Cell)
	convoName := fmt.Sprintf("%s %s", su.NameFirst, su.NameLast[0:1])
	convoId := ""
	existing, err := t.findConversationsByNumber(toNum)
	if err != nil {
		return fmt.Errorf("findConversationsByNumber: %w", err)
	}

	// Create a new conversation if none exists
	if len(existing) == 0 {
		convoId, err = t.addNumberToConversation(toNum, convoName)
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
		convoId = *existing[0].ConversationSid
	}

	// Send Opt-in confirmation
	if err := t.optInConfirmation(ctx, toNum); err != nil {
		return fmt.Errorf("optInConfirmation: %w", err)
	}

	// create user-specific info session details URL
	msgngURL, err := su.shortMessagingURL(os.Getenv("GREENLIGHT_HOST"), os.Getenv("OS_RENDERING_SERVICE_URL"))
	if err != nil {
		return fmt.Errorf("shortMessagingURL: %w", err)
	}

	shorty := NewURLShortener(ShortenerOpts{apiKey: os.Getenv("URL_SHORTENER_API_KEY")})

	shortLink, err := shorty.ShortenURL(ctx, msgngURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "shortenURL ERROR: %v", err)
		// Don't early return. ShortenURL returns the original URL if there is a failure
		// Fallback to long URL if shortener fails
	}

	// Create the SMS message body
	msg, err := su.shortMessage(shortLink)
	if err != nil {
		return fmt.Errorf("shortMessage: %w", err)
	}

	err = t.sendSMSInConversation(msg, convoId)
	if err != nil {
		return fmt.Errorf("sendSMS: %w", err)
	}

	err = t.sendConvoWebhook(ctx, convoId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sendConvoWebhook (messenger API): %v", err)
	}
	// Carry on even if the Messenger API webhook fails
	return nil
}

func (t *smsService) name() string {
	return "twilio service"
}

// SendSMSInConversation uses the Twilio Conversations API to send a message to a specific Conversation. Twilio will then broadcast the message to the Conversation participants. In our case, this is two SMS-capable phone numbers.
func (t *smsService) sendSMSInConversation(body string, convoId string) error {
	params := &conversations.CreateServiceConversationMessageParams{
		Body:   &body,
		Author: &t.conversationsIdentity,
	}

	_, err := t.client.ConversationsV1.CreateServiceConversationMessage(t.conversationsSid, convoId, params)
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

	req, err := http.NewRequest(http.MethodPost, url, nil)
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
	convoId := ""
	existing, err := t.findConversationsByNumber(toNum)
	if err != nil {
		return fmt.Errorf("findConversationsByNumber: %w", err)
	}

	// Create a new conversation if none exists
	// I think we should always already have an existing conversation
	if len(existing) == 0 {
		convoId, err = t.addNumberToConversation(toNum, toNum)
		if err != nil {
			return fmt.Errorf("addNumberToConversation: %w", err)
		}
	} else {
		if len(existing) > 1 {
			return fmt.Errorf("found more than one existing conversation for cell: %q. %+v", toNum, existing)
		}
		// There can only be one..
		convoId = *existing[0].ConversationSid
	}
	err = t.sendSMSInConversation(msg, convoId)
	if err != nil {
		return fmt.Errorf("sendSMS: %w", err)
	}
	return nil
}
