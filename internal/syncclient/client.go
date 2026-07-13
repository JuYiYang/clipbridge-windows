package syncclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/JuYiYang/clipbridge-windows/internal/protocol"
)

type Client struct {
	baseURL    string
	token      string
	deviceID   string
	httpClient *http.Client
}

func New(baseURL string, token string, deviceID string) Client {
	return Client{
		baseURL:  baseURL,
		token:    token,
		deviceID: deviceID,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c Client) PushItems(ctx context.Context, items []protocol.ClipboardItem) (protocol.PushResponse, error) {
	var response protocol.PushResponse
	payload := protocol.PushRequest{
		DeviceID: c.deviceID,
		Items:    items,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return response, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/v1/clipboard/items", bytes.NewReader(body))
	if err != nil {
		return response, err
	}
	req.Header.Set("Content-Type", "application/json")

	if err := c.do(req, &response); err != nil {
		return response, err
	}
	return response, nil
}

func (c Client) PullItems(ctx context.Context, since float64) (protocol.PullResponse, error) {
	var response protocol.PullResponse
	path := "/v1/clipboard/items?since=" + url.QueryEscape(strconv.FormatFloat(since, 'f', -1, 64))
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return response, err
	}

	if err := c.do(req, &response); err != nil {
		return response, err
	}
	return response, nil
}

func (c Client) newRequest(ctx context.Context, method string, path string, body *bytes.Reader) (*http.Request, error) {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = body
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-ClipBridge-Device-ID", c.deviceID)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return req, nil
}

func (c Client) do(req *http.Request, target any) error {
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("clipbridge server returned HTTP %d", res.StatusCode)
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(target)
}
