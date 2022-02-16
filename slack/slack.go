package slack

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type Message struct {
	Text string `json:"text"`
}

func SendWebhook(url string, msg Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	lw := LogWriter{}
	io.Copy(lw, resp.Body)
	return nil
}
