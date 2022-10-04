package signup

import (
	"encoding/json"
	"fmt"

	"github.com/twilio/twilio-go"
	twilioAPI "github.com/twilio/twilio-go/rest/api/v2010"
)

type smsService struct {
	// Client for making requests to Twilio's API.
	client *twilio.RestClient
	// Phone number SMS messages are sent from.
	fromPhoneNum string
}

func NewTwilioService(accountSid, authToken, fromPhoneNum string) *smsService {
	return &smsService{
		client: twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: accountSid,
			Password: authToken,
		}),
		fromPhoneNum: fromPhoneNum,
	}
}

func (t *smsService) run(su Signup) error {
	// Call [Peter's service] to get the custom messaging page URL.
	// Create the SMS message body
	// Send the SMS
	return t.sendSMS(su)
}

func (t *smsService) name() string {
	return "twilio service"
}

func (t *smsService) sendSMS(su Signup) error {
	msg := fmt.Sprintf("Thanks for signing up %s!", su.NameFirst)

	params := &twilioAPI.CreateMessageParams{}
	params.SetTo(su.Cell)
	params.SetFrom(t.fromPhoneNum)
	params.SetBody(msg)

	resp, err := t.client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("%s: sendSMS: createMessage: %v", t.name(), err)
	}

	response, _ := json.Marshal(*resp)
	fmt.Printf("Twilio: %s", prettyPrint(response))
	return nil
}
