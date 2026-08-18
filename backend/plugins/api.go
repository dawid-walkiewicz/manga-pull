package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
)

type HTTPResponse struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"-"`
}

func (r HTTPResponse) Text() string {
	return string(r.Body)
}

func (r HTTPResponse) Bytes() []byte {
	return r.Body
}

func (r HTTPResponse) JSON() (any, error) {
	var value any

	if err := json.Unmarshal(r.Body, &value); err != nil {
		return nil, err
	}

	return value, nil
}

type PluginAPIClient struct {
	client  *http.Client
	domains []string
}

func NewPluginAPIClient(plugin *Plugin) *PluginAPIClient {
	return &PluginAPIClient{
		client:  &http.Client{},
		domains: plugin.Domains,
	}
}

func (c *PluginAPIClient) Get(rawURL string) (map[string]any, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if !slices.Contains(c.domains, u.Hostname()) {
		return nil, fmt.Errorf(
			"domain %q is not allowed",
			u.Hostname(),
		)
	}

	resp, err := c.client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := HTTPResponse{
		Status:  resp.StatusCode,
		Headers: resp.Header,
		Body:    body,
	}

	return map[string]any{
		"status":  response.Status,
		"headers": response.Headers,
		"text":    response.Text,
		"json":    response.JSON,
		"bytes":   response.Bytes,
	}, nil
}
