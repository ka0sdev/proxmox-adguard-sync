package adguard

import (
	"bytes"
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

type RewriteUpdate struct {
	Target Rewrite `json:"target"`
	Update Rewrite `json:"update"`
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

	if err := c.request(
		ctx,
		http.MethodGet,
		"/control/rewrite/list",
		nil,
		&rewrites,
	); err != nil {
		return nil, fmt.Errorf(
			"list AdGuard DNS rewrites: %w",
			err,
		)
	}

	return rewrites, nil
}

func (c *Client) AddRewrite(
	ctx context.Context,
	rewrite Rewrite,
) error {
	if err := validateRewrite(rewrite); err != nil {
		return fmt.Errorf(
			"validate rewrite to add: %w",
			err,
		)
	}

	if err := c.request(
		ctx,
		http.MethodPost,
		"/control/rewrite/add",
		rewrite,
		nil,
	); err != nil {
		return fmt.Errorf(
			"add AdGuard DNS rewrite %q: %w",
			rewrite.Domain,
			err,
		)
	}

	return nil
}

func (c *Client) UpdateRewrite(
	ctx context.Context,
	current Rewrite,
	desired Rewrite,
) error {
	if err := validateRewrite(current); err != nil {
		return fmt.Errorf(
			"validate current rewrite: %w",
			err,
		)
	}

	if err := validateRewrite(desired); err != nil {
		return fmt.Errorf(
			"validate desired rewrite: %w",
			err,
		)
	}

	payload := RewriteUpdate{
		Target: current,
		Update: desired,
	}

	if err := c.request(
		ctx,
		http.MethodPut,
		"/control/rewrite/update",
		payload,
		nil,
	); err != nil {
		return fmt.Errorf(
			"update AdGuard DNS rewrite %q: %w",
			current.Domain,
			err,
		)
	}

	return nil
}

func (c *Client) DeleteRewrite(
	ctx context.Context,
	rewrite Rewrite,
) error {
	if err := validateRewrite(rewrite); err != nil {
		return fmt.Errorf(
			"validate rewrite to delete: %w",
			err,
		)
	}

	if err := c.request(
		ctx,
		http.MethodPost,
		"/control/rewrite/delete",
		rewrite,
		nil,
	); err != nil {
		return fmt.Errorf(
			"delete AdGuard DNS rewrite %q: %w",
			rewrite.Domain,
			err,
		)
	}

	return nil
}

func (c *Client) request(
	ctx context.Context,
	method string,
	path string,
	payload any,
	destination any,
) error {
	var body io.Reader

	if payload != nil {
		content, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf(
				"encode AdGuard request: %w",
				err,
			)
		}

		body = bytes.NewReader(content)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		method,
		c.resolvePath(path),
		body,
	)
	if err != nil {
		return fmt.Errorf(
			"create AdGuard request: %w",
			err,
		)
	}

	request.Header.Set("Accept", "application/json")
	request.SetBasicAuth(c.username, c.password)

	if payload != nil {
		request.Header.Set(
			"Content-Type",
			"application/json",
		)
	}

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
		responseBody, _ := io.ReadAll(
			io.LimitReader(response.Body, 4096),
		)

		return fmt.Errorf(
			"AdGuard returned HTTP %d: %s",
			response.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	if destination == nil {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
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

	basePath := strings.TrimRight(
		resolved.Path,
		"/",
	)

	requestPath := "/" + strings.TrimLeft(
		path,
		"/",
	)

	if strings.HasSuffix(basePath, "/control") &&
		strings.HasPrefix(
			requestPath,
			"/control/",
		) {
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

func validateRewrite(rewrite Rewrite) error {
	if strings.TrimSpace(rewrite.Domain) == "" {
		return errors.New(
			"rewrite domain must not be empty",
		)
	}

	if strings.TrimSpace(rewrite.Answer) == "" {
		return errors.New(
			"rewrite answer must not be empty",
		)
	}

	return nil
}
