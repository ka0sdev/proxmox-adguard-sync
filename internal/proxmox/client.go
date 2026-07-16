package proxmox

import (
	"context"
	"crypto/tls"
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

type Client struct {
	baseURL     *url.URL
	tokenID     string
	tokenSecret string
	httpClient  *http.Client
}

type ClientOptions struct {
	BaseURL     string
	TokenID     string
	TokenSecret string
	VerifyTLS   bool
	Timeout     time.Duration
}

func NewClient(options ClientOptions) (*Client, error) {
	baseURL, err := parseBaseURL(options.BaseURL)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(options.TokenID) == "" {
		return nil, errors.New("Proxmox token ID must not be empty")
	}

	if strings.TrimSpace(options.TokenSecret) == "" {
		return nil, errors.New("Proxmox token secret must not be empty")
	}

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: !options.VerifyTLS, //nolint:gosec
	}

	return &Client{
		baseURL:     baseURL,
		tokenID:     strings.TrimSpace(options.TokenID),
		tokenSecret: strings.TrimSpace(options.TokenSecret),
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}, nil
}

func (c *Client) Version(ctx context.Context) (Version, error) {
	var version Version

	if err := c.get(ctx, "/version", &version); err != nil {
		return Version{}, fmt.Errorf("get Proxmox version: %w", err)
	}

	return version, nil
}

func (c *Client) ListGuests(ctx context.Context) ([]Guest, error) {
	var resources []Guest

	if err := c.get(ctx, "/cluster/resources", &resources); err != nil {
		return nil, fmt.Errorf("list cluster resources: %w", err)
	}

	guests := make([]Guest, 0, len(resources))

	for _, resource := range resources {
		switch resource.Type {
		case GuestTypeQEMU, GuestTypeLXC:
			guests = append(guests, resource)
		}
	}

	return guests, nil
}

func (c *Client) get(
	ctx context.Context,
	path string,
	result any,
) error {
	requestURL := c.resolveURL(path)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		requestURL,
		nil,
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	request.Header.Set(
		"Authorization",
		fmt.Sprintf(
			"PVEAPIToken=%s=%s",
			c.tokenID,
			c.tokenSecret,
		),
	)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "proxmox-adguard-sync")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		return decodeAPIError(response)
	}

	var envelope apiResponse[json.RawMessage]

	decoder := json.NewDecoder(
		io.LimitReader(response.Body, 10<<20),
	)

	if err := decoder.Decode(&envelope); err != nil {
		return fmt.Errorf("decode response envelope: %w", err)
	}

	if err := json.Unmarshal(envelope.Data, result); err != nil {
		return fmt.Errorf("decode response data: %w", err)
	}

	return nil
}

func (c *Client) resolveURL(path string) string {
	base := strings.TrimRight(c.baseURL.String(), "/")
	requestPath := "/" + strings.TrimLeft(path, "/")

	return base + requestPath
}

func parseBaseURL(value string) (*url.URL, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("Proxmox base URL must not be empty")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("parse Proxmox base URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf(
			"Proxmox base URL scheme must be http or https",
		)
	}

	if parsed.Host == "" {
		return nil, errors.New(
			"Proxmox base URL must contain a host",
		)
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return parsed, nil
}

func decodeAPIError(response *http.Response) error {
	const maximumErrorBodySize = 1 << 20

	body, err := io.ReadAll(
		io.LimitReader(response.Body, maximumErrorBodySize),
	)
	if err != nil {
		return fmt.Errorf(
			"Proxmox API returned %s and its response body could not be read: %w",
			response.Status,
			err,
		)
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(response.StatusCode)
	}

	return fmt.Errorf(
		"Proxmox API returned %s: %s",
		response.Status,
		message,
	)
}
