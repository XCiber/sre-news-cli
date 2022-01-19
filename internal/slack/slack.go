package slack

import (
	"bytes"
	"errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"time"
)

type SlackOptions struct {
	WebHook string
}

type SlackClient struct {
	options SlackOptions
	client  *http.Client
}

func (sc *SlackClient) Send(message string) error {
	if message == "" {
		return errors.New("empty message")
	}
	request, err := http.NewRequest(http.MethodPost, sc.options.WebHook, bytes.NewBuffer([]byte(message)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	response, err := sc.client.Do(request)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Debug("response body reader close: ", err)
		}
	}(response.Body)

	return nil
}

func New(webhook string) *SlackClient {
	return &SlackClient{
		options: SlackOptions{
			WebHook: webhook,
		},
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}
