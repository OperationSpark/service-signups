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
		// Messaging service SID
		messagingServiceSid string
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
		messagingServiceSid        string
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

	return &smsService{
		apiBase: apiBase,
		client: twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: o.accountSID,
			Password: o.authToken,
			Client:   o.client,
		}),
		fromPhoneNum:               o.fromPhoneNum,
		messagingServiceSid:        o.messagingServiceSid,
		opSparkMessagingSvcBaseURL: smsBaseURL,
		conversationsSid:           o.conversationsSid,
		conversationsIdentity:      o.conversationsIdentity,
	}
}

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

	shorty := NewURLShortener("https://ospk.org", os.Getenv("URL_SHORTENER_API_KEY"))
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

func (t *smsService) sendSMSInConversation(body string, convoId string) error {
	params := &conversations.CreateServiceConversationMessageParams{Body: &body}

	resp, err := t.client.ConversationsV1.CreateServiceConversationMessage(t.conversationsSid, convoId, params)
	if err != nil {
		return fmt.Errorf("createServiceConversationMessage: %v", err)
	}

	fmt.Println(resp)
	return nil
}

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

func (t *smsService) addNumberToConversation(phNum, friendlyName string) (string, error) {
	cp := &conversations.CreateConversationParams{}
	cp.SetFriendlyName(friendlyName)

	// create new convo
	cResp, err := t.client.ConversationsV1.CreateConversation(cp)
	if err != nil {
		return "", fmt.Errorf("createConversation: %v", err)
	}
	fmt.Println(cResp)

	// Add Operation Spark Conversation Identity
	ppp := &conversations.CreateConversationParticipantParams{}
	ppp.SetIdentity(t.conversationsIdentity)
	respUser, err := t.client.ConversationsV1.CreateConversationParticipant(*cResp.Sid, ppp)
	if err != nil {
		return "", fmt.Errorf("createConversationParticipant with Identity: %v", err)
	}

	fmt.Println(respUser)

	// Add SMS Recipient to conversation
	pp := &conversations.CreateConversationParticipantParams{}
	pp.SetMessagingBindingAddress(phNum)
	pp.SetMessagingBindingProxyAddress(t.fromPhoneNum)

	_, err = t.client.ConversationsV1.CreateConversationParticipant(*cResp.Sid, pp)
	if err != nil {
		return "", fmt.Errorf("createConversationParticipant: %v", err)
	}

	return *cResp.Sid, nil
}

func (t *smsService) formatCell(cell string) string {
	return "+1" + strings.ReplaceAll(cell, "-", "")
}
