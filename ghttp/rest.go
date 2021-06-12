package ghttp

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"net/http"
)

type RequestOption func(req *http.Request)

type HTTPClient struct {
	MaxRead int64
	client  *http.Client
}

var DefaultClient = NewHTTPClient(nil)

func NewHTTPClient(client *http.Client) *HTTPClient {
	if client == nil {
		client = http.DefaultClient
	}

	return &HTTPClient{
		MaxRead: 10 * 1024 * 1024,
		client:  client,
	}
}

func WithHeader(key, value string) RequestOption {
	return func(req *http.Request) {
		if key == "" || value == "" {
			return
		}

		req.Header.Set(key, value)
	}
}

func WithBasicAuth(username, password string) RequestOption {
	return func(req *http.Request) {
		req.SetBasicAuth(username, password)
	}
}

func (c *HTTPClient) DoGetJSON(url string, resObj interface{}, opts ...RequestOption) error {
	res, err := c.DoGet(url, opts...)
	if err != nil {
		return NewError(-1, res, errors.WithStack(err))
	}

	if err := json.Unmarshal(res, resObj); err != nil {
		return NewError(-1, res, errors.WithStack(err))
	}
	return nil
}

func (c *HTTPClient) DoPostJSON(url string, reqObj interface{}, resObj interface{}, opts ...RequestOption) error {
	var body []byte
	var err error
	if reqObj != nil {
		body, err = json.Marshal(reqObj)
		if err != nil {
			return NewError(-1, nil, errors.WithStack(err))
		}
	}

	res, err := c.DoPost(url, body, append([]RequestOption{
		WithHeader("Content-Type", "application/json"),
	}, opts...)...)
	if err != nil {
		return err
	}

	if resObj == nil {
		return nil
	}

	if err := json.Unmarshal(res, resObj); err != nil {
		return NewError(-1, res, errors.WithStack(err))
	}
	return nil
}

func (c *HTTPClient) DoGet(url string, opts ...RequestOption) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, NewError(-1, nil, errors.WithStack(err))
	}
	return c.doReq(req, opts...)
}

func (c *HTTPClient) DoPost(url string, body []byte, opts ...RequestOption) ([]byte, error) {
	bodyR := bytes.NewReader(body)
	req, err := http.NewRequest("POST", url, bodyR)
	if err != nil {
		return nil, NewError(-1, nil, errors.WithStack(err))
	}
	return c.doReq(req, opts...)
}

func (c *HTTPClient) doReq(req *http.Request, opts ...RequestOption) ([]byte, error) {
	client := &http.Client{}
	for _, opt := range opts {
		opt(req)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, NewError(-1, nil, errors.WithStack(err))
	}
	defer res.Body.Close()

	if res.StatusCode == 204 {
		return nil, nil
	}

	resBody, err := io.ReadAll(io.LimitReader(res.Body, c.MaxRead))
	if err != nil {
		return nil, NewError(-1, nil, errors.WithStack(err))
	}

	if res.StatusCode < 200 || res.StatusCode > 300 {
		return nil, NewError(res.StatusCode, resBody, errors.Errorf("non-200 status code %d", res.StatusCode))
	}

	return resBody, nil
}
