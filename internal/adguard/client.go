package adguard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultRequestTimeout = 15 * time.Second

type Rewrite struct {
	Domain  string `json:"domain"`
	Answer  string `json:"answer"`
	Enabled *bool  `json:"enabled,omitempty"`
}

type ClientOptions struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient *http.Client
}

type Client struct {
	baseURL    *url.URL
	username   string
	password   string
	httpClient *http.Client
}

func NewClient(options ClientOptions) (*Client, error) {
	baseURL := strings.TrimRight(
		strings.TrimSpace(options.BaseURL),
		"/",
	)

	if baseURL == "" {
		return nil, errors.New(
			"AdGuard base URL must not be empty",
		)
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf(
			"parse AdGuard base URL: %w",
			err,
		)
	}

	if parsedURL.Scheme != "http" &&
		parsedURL.Scheme != "https" {
		return nil, fmt.Errorf(
			"unsupported AdGuard URL scheme %q",
			parsedURL.Scheme,
		)
	}

	if parsedURL.Host == "" {
		return nil, errors.New(
			"AdGuard base URL must include a host",
		)
	}

	username := strings.TrimSpace(options.Username)
	if username == "" {
		return nil, errors.New(
			"AdGuard username must not be empty",
		)
	}

	if options.Password == "" {
		return nil, errors.New(
			"AdGuard password must not be empty",
		)
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultRequestTimeout,
		}
	}

	return &Client{
		baseURL:    parsedURL,
		username:   username,
		password:   options.Password,
		httpClient: httpClient,
	}, nil
}

func (c *Client) ListRewrites(
	ctx context.Context,
) ([]Rewrite, error) {
	var rewrites []Rewrite

	if err := c.get(
		ctx,
		"/control/rewrite/list",
		&rewrites,
	); err != nil {
		return nil, fmt.Errorf(
			"list AdGuard DNS rewrites: %w",
			err,
		)
	}

	return rewrites, nil
}

func (c *Client) get(
	ctx context.Context,
	path string,
	destination any,
) error {
	requestURL := c.resolvePath(path)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		requestURL,
		nil,
	)
	if err != nil {
		return fmt.Errorf(
			"create AdGuard request: %w",
			err,
		)
	}

	request.Header.Set("Accept", "application/json")
	request.SetBasicAuth(c.username, c.password)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf(
			"execute AdGuard request: %w",
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		body, _ := io.ReadAll(
			io.LimitReader(response.Body, 4096),
		)

		return fmt.Errorf(
			"AdGuard returned HTTP %d: %s",
			response.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	if err := json.NewDecoder(response.Body).Decode(
		destination,
	); err != nil {
		return fmt.Errorf(
			"decode AdGuard response: %w",
			err,
		)
	}

	return nil
}

func (c *Client) resolvePath(path string) string {
	resolved := *c.baseURL

	basePath := strings.TrimRight(resolved.Path, "/")
	requestPath := "/" + strings.TrimLeft(path, "/")

	if strings.HasSuffix(basePath, "/control") &&
		strings.HasPrefix(requestPath, "/control/") {
		requestPath = strings.TrimPrefix(
			requestPath,
			"/control",
		)
	}

	resolved.Path = basePath + requestPath
	resolved.RawQuery = ""
	resolved.Fragment = ""

	return resolved.String()
}
