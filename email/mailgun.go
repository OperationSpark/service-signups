package email

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mailgun/mailgun-go/v4"
)

var domain = os.Getenv("MAIL_DOMAIN")
var privateApiKey = os.Getenv("MAIL_GUN_PRIVATE_API_KEY")

type Message struct {
	recipient string
	sender    string
	subject   string
	body      string
}

func SendWelcome(to string) error {
	msg := Message{
		recipient: to,
		sender:    fmt.Sprintf("info@%s", domain),
		subject:   "Welcome from Operation Spark!",
		body:      "this is a test", // TODO Welcome Email HTML here
	}
	resp, err := SendSimpleMessage(domain, privateApiKey, &msg)
	fmt.Println(resp)

	if err != nil {
		return err
	}
	return nil
}

func SendSimpleMessage(domain, apiKey string, msg *Message) (string, error) {
	mg := mailgun.NewMailgun(domain, apiKey)

	message := mg.NewMessage(msg.sender, msg.subject, msg.body, msg.recipient)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Send the message with a 10 second timeout
	resp, id, err := mg.Send(ctx, message)

	if err != nil {
		return "", err
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)
	return resp, nil
}
