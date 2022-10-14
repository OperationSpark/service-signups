package signup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/twilio/twilio-go"
	"github.com/twilio/twilio-go/client"
	twilioAPI "github.com/twilio/twilio-go/rest/api/v2010"
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
	}

	twilioServiceOptions struct {
		accountSID                 string
		authToken                  string
		client                     client.BaseClient
		fromPhoneNum               string
		messagingServiceSid        string
		opSparkMessagingSvcBaseURL string
		apiBase                    string
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
	}
}

func (t *smsService) run(ctx context.Context, su Signup) error {
	// Call [Peter's service] to get the custom messaging page URL.
	// Create the SMS message body
	// Send the SMS
	return t.sendSMS(su)
}

func (t *smsService) name() string {
	return "twilio service"
}

func (t *smsService) sendSMS(su Signup) error {
	mgsngURL, err := su.shortMessagingURL(t.opSparkMessagingSvcBaseURL)
	if err != nil {
		return fmt.Errorf("shortMessagingURL: %v", err)
	}

	msg, err := su.shortMessage(mgsngURL)
	if err != nil {
		return fmt.Errorf("shortMessage: %v", err)
	}

	params := &twilioAPI.CreateMessageParams{}
	params.SetTo(su.Cell)
	params.SetFrom(t.fromPhoneNum)
	params.SetBody(msg)
	params.SetMessagingServiceSid(t.messagingServiceSid)

	// ** The following is a temporary work around to use the "ShortenUrls" param that is not yet supported by this SDK.
	// I will put in a PR to add support. If accepted, merged, and released, we can delete these lines, and just use CreateMessage(params).
	// https://github.com/twilio/twilio-go/issues/194
	endpoint := fmt.Sprintf("%s/2010-04-01/Accounts/%s/Messages.json", t.apiBase, t.client.Client.AccountSid())
	data := url.Values{
		"To":                  []string{*params.To},
		"From":                []string{*params.From},
		"Body":                []string{*params.Body},
		"MessagingServiceSid": []string{*params.MessagingServiceSid},
		"ShortenUrls":         []string{"true"},
	}
	headers := make(map[string]interface{})
	resp, err := t.client.Post(endpoint, data, headers)
	if err != nil {
		return fmt.Errorf("%s: sendSMS: createMessage: %v", t.name(), err)
	}
	// ** End of workaround ** //

	// Comment back in if/when ShortenUrls param is supported by the SDK
	// params.SetShortenUrls(true)
	// resp, err := t.client.Api.CreateMessage(params)

	response, _ := json.Marshal(resp.Body)
	fmt.Printf("Twilio CreateMessage response: %s", prettyPrint(response))
	return nil
}
