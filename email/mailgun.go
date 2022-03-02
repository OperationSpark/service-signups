package email

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var BASE_URL = "https://api.mailgun.net/"

type Message struct {
	from    string
	to      string
	cc      string
	bcc     string
	subject string
	text    string
	html    string
}

func (m *Message) form() url.Values {
	data := url.Values{}
	data.Set("from", "info@operationspark.org")
	data.Set("to", m.to)
	data.Set("subject", m.subject)
	data.Set("text", m.text)
	return data
}

func Send(msg Message) error {
	DOMAIN, ok := os.LookupEnv("MAIL_DOMAIN")
	if !ok {
		return errors.New("'MAIL_DOMAIN' not set")
	}
	url := fmt.Sprintf("https://%sv3/%s/messages", BASE_URL, DOMAIN)
	data := msg.form()

	client := &http.Client{}
	resp, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	resp.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	res, err := client.Do(resp)
	if err != nil {
		return err
	}
	fmt.Println(res.Status)
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}
