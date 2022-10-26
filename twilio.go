package signup

import (
	"context"
	"fmt"
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
		// Base URL for OpSpark Messaging Service.
		// Default: https://sms.operationspark.org
		opSparkMessagingSvcBaseURL string
		conversationsSid           string
		// Twilio Conversations Service User identity name.
		conversationsIdentity string
	}

	twilioServiceOptions struct {
		accountSID                 string
		authToken                  string
		client                     client.BaseClient
		fromPhoneNum               string
		opSparkMessagingSvcBaseURL string
		apiBase                    string
		conversationsSid           string
		conversationsIdentity      string
	}
)

func NewTwilioService(o twilioServiceOptions) *smsService {
	smsBaseURL := "https://sms.operationspark.org"
	if len(o.opSparkMessagingSvcBaseURL) > 0 {
		smsBaseURL = o.opSparkMessagingSvcBaseURL
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
		opSparkMessagingSvcBaseURL: smsBaseURL,
		conversationsSid:           o.conversationsSid,
		conversationsIdentity:      conversationsIdentity,
	}
}

// Run sends an Info Session signup confirmation SMS to the registered participant. We use Twilio's Conversations API instead of the Messaging API to allow multiple staff members communicate with the participant through the same outgoing SMS number.
// The confirmation SMS contains a link that when clicked generates a custom page containing information on the upcoming Info Session. This signup-specific link is shortened before sent.
//
// Note: Twilio has a free Link Shortening service, but it is only available with the Messaging API, not Conversations.
func (t *smsService) run(ctx context.Context, su Signup) error {
	toNum := t.formatCell(su.Cell)
	convoName := fmt.Sprintf("%s %s", su.NameFirst, su.NameLast[0:1])
	convoId := ""
	existing, err := t.findConversationsByNumber(toNum)
	if err != nil {
		return fmt.Errorf("findConversationsByNumber: %v", err)
	}

	// Create a new conversation if none exists
	if len(existing) == 0 {
		convoId, err = t.addNumberToConversation(toNum, convoName)
		if err != nil {
			return fmt.Errorf("addNumberToConversation: %v", err)
		}
	} else {
		// TODO: Fix this potentially faulty logic if picking the first existing conversation
		convoId = *existing[0].ConversationSid
	}

	mgsngURL, err := su.shortMessagingURL(t.opSparkMessagingSvcBaseURL)
	if err != nil {
		return fmt.Errorf("shortMessagingURL: %v", err)
	}

	shorty := NewURLShortener(ShortenerOpts{apiKey: os.Getenv("URL_SHORTENER_API_KEY")})

	shortLink, err := shorty.ShortenURL(ctx, mgsngURL)
	if err != nil {
		return fmt.Errorf("shortenURL: %v", err)
	}

	// Create the SMS message body
	msg, err := su.shortMessage(shortLink)
	if err != nil {
		return fmt.Errorf("shortMessage: %v", err)
	}

	err = t.sendSMSInConversation(msg, convoId)
	if err != nil {
		return fmt.Errorf("sendSMS: %v", err)
	}
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
		return fmt.Errorf("createServiceConversationMessage: %v", err)
	}

	return nil
}

// FindConversationsByNumber finds all Twilio Conversations that have the given phone number as a participant.
func (t *smsService) findConversationsByNumber(phNum string) ([]conversations.ConversationsV1ParticipantConversation, error) {
	params := &conversations.ListParticipantConversationParams{}
	params.SetAddress(phNum)
	params.SetLimit(20)

	resp, err := t.client.ConversationsV1.ListParticipantConversation(params)
	if err != nil {
		return resp, fmt.Errorf("listParticipantConversation: %v", err)
	}
	return resp, nil
}

// AddNumberToConversation creates a new Conversation and adds two participants - the Operation Spark Service Identity ("services@operationspark.org"), and the SMS recipient's phone number.
func (t *smsService) addNumberToConversation(phNum, friendlyName string) (string, error) {
	cp := &conversations.CreateConversationParams{}
	cp.SetFriendlyName(friendlyName)

	// Create new Conversation
	cResp, err := t.client.ConversationsV1.CreateConversation(cp)
	if err != nil {
		return "", fmt.Errorf("createConversation: %v", err)
	}

	// Add Operation Spark Conversation Identity
	ppp := &conversations.CreateConversationParticipantParams{}
	ppp.SetIdentity(t.conversationsIdentity)
	_, err = t.client.ConversationsV1.CreateConversationParticipant(*cResp.Sid, ppp)
	if err != nil {
		return "", fmt.Errorf("createConversationParticipant with Identity: %v", err)
	}

	// Add SMS Recipient to conversation
	pp := &conversations.CreateConversationParticipantParams{}
	pp.SetMessagingBindingAddress(phNum)
	pp.SetMessagingBindingProxyAddress(t.fromPhoneNum)
	friendlyNameWithNum := fmt.Sprintf("%s (%s)", friendlyName, phNum)
	pp.SetAttributes(fmt.Sprintf(`{"friendlyName": %q}`, friendlyNameWithNum))

	_, err = t.client.ConversationsV1.CreateConversationParticipant(*cResp.Sid, pp)
	if err != nil {
		return "", fmt.Errorf("createConversationParticipant: %v", err)
	}

	return *cResp.Sid, nil
}

// FormatCell prepends the US country code, "+1", and removes any dashes from a phone number string.
func (t *smsService) formatCell(cell string) string {
	return "+1" + strings.ReplaceAll(cell, "-", "")
}
