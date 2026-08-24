package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
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

type HTTPRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type PluginAPIClient struct {
	client   *http.Client
	domains  []string
	pluginID string
}

func NewPluginAPIClient(plugin *Plugin) *PluginAPIClient {
	return &PluginAPIClient{
		client:   &http.Client{},
		domains:  plugin.Domains,
		pluginID: plugin.ID,
	}
}

func (c *PluginAPIClient) Request(req HTTPRequest) (map[string]any, error) {
	u, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}

	if u.Hostname() == "" {
		return nil, fmt.Errorf("missing URL host")
	}

	if !slices.Contains(c.domains, u.Hostname()) {
		return nil, fmt.Errorf("domain %q is not allowed", u.Hostname())
	}

	method := req.Method
	if method == "" {
		method = http.MethodGet
	}

	var body io.Reader
	if req.Body != "" {
		body = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequest(method, req.URL, body)
	if err != nil {
		return nil, err
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := HTTPResponse{
		Status:  resp.StatusCode,
		Headers: resp.Header,
		Body:    data,
	}

	return map[string]any{
		"status":  response.Status,
		"headers": response.Headers,
		"text":    response.Text,
		"json":    response.JSON,
		"bytes":   response.Bytes,
	}, nil
}

func (c *PluginAPIClient) Log(level, message string, data any) {
	log.Printf(
		"plugin=%s level=%s message=%q data=%v",
		c.pluginID,
		level,
		message,
		data,
	)
}
