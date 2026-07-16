package proxmox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVersion(t *testing.T) {
	const (
		expectedTokenID     = "sync@pve!adguard-sync"
		expectedTokenSecret = "test-secret"
	)

	server := httptest.NewServer(
		http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			if request.Method != http.MethodGet {
				t.Errorf(
					"request method = %q, expected %q",
					request.Method,
					http.MethodGet,
				)
			}

			if request.URL.Path != "/api2/json/version" {
				t.Errorf(
					"request path = %q, expected %q",
					request.URL.Path,
					"/api2/json/version",
				)
			}

			expectedAuthorization := "PVEAPIToken=" +
				expectedTokenID +
				"=" +
				expectedTokenSecret

			if authorization := request.Header.Get(
				"Authorization",
			); authorization != expectedAuthorization {
				t.Errorf(
					"Authorization header = %q, expected %q",
					authorization,
					expectedAuthorization,
				)
			}

			if accept := request.Header.Get(
				"Accept",
			); accept != "application/json" {
				t.Errorf(
					"Accept header = %q, expected application/json",
					accept,
				)
			}

			writer.Header().Set(
				"Content-Type",
				"application/json",
			)

			_, _ = writer.Write([]byte(`{
				"data": {
					"version": "9.1.11",
					"release": "9.1",
					"repoid": "test-repository"
				}
			}`))
		}),
	)
	defer server.Close()

	client, err := NewClient(ClientOptions{
		BaseURL:     server.URL + "/api2/json/",
		TokenID:     expectedTokenID,
		TokenSecret: expectedTokenSecret,
		VerifyTLS:   true,
	})
	if err != nil {
		t.Fatalf("NewClient() returned an unexpected error: %v", err)
	}

	version, err := client.Version(context.Background())
	if err != nil {
		t.Fatalf("Version() returned an unexpected error: %v", err)
	}

	if version.Version != "9.1.11" {
		t.Errorf(
			"Version.Version = %q, expected %q",
			version.Version,
			"9.1.11",
		)
	}

	if version.Release != "9.1" {
		t.Errorf(
			"Version.Release = %q, expected %q",
			version.Release,
			"9.1",
		)
	}

	if version.RepoID != "test-repository" {
		t.Errorf(
			"Version.RepoID = %q, expected %q",
			version.RepoID,
			"test-repository",
		)
	}
}

func TestVersionReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			writer http.ResponseWriter,
			_ *http.Request,
		) {
			http.Error(
				writer,
				"authentication failed",
				http.StatusUnauthorized,
			)
		}),
	)
	defer server.Close()

	client, err := NewClient(ClientOptions{
		BaseURL:     server.URL,
		TokenID:     "sync@pve!adguard-sync",
		TokenSecret: "invalid-secret",
		VerifyTLS:   true,
	})
	if err != nil {
		t.Fatalf("NewClient() returned an unexpected error: %v", err)
	}

	_, err = client.Version(context.Background())
	if err == nil {
		t.Fatal("Version() returned nil error, expected API error")
	}

	expectedValues := []string{
		"401 Unauthorized",
		"authentication failed",
	}

	for _, expectedValue := range expectedValues {
		if !strings.Contains(err.Error(), expectedValue) {
			t.Errorf(
				"error = %q, expected it to contain %q",
				err,
				expectedValue,
			)
		}
	}
}

func TestNewClientAppliesDefaultTimeout(t *testing.T) {
	client, err := NewClient(ClientOptions{
		BaseURL:     "https://proxmox.example.com/api2/json",
		TokenID:     "sync@pve!adguard-sync",
		TokenSecret: "test-secret",
		VerifyTLS:   true,
	})
	if err != nil {
		t.Fatalf("NewClient() returned an unexpected error: %v", err)
	}

	if client.httpClient.Timeout != defaultRequestTimeout {
		t.Errorf(
			"HTTP timeout = %s, expected %s",
			client.httpClient.Timeout,
			defaultRequestTimeout,
		)
	}
}

func TestNewClientAppliesCustomTimeout(t *testing.T) {
	const expectedTimeout = 30 * time.Second

	client, err := NewClient(ClientOptions{
		BaseURL:     "https://proxmox.example.com/api2/json",
		TokenID:     "sync@pve!adguard-sync",
		TokenSecret: "test-secret",
		VerifyTLS:   true,
		Timeout:     expectedTimeout,
	})
	if err != nil {
		t.Fatalf("NewClient() returned an unexpected error: %v", err)
	}

	if client.httpClient.Timeout != expectedTimeout {
		t.Errorf(
			"HTTP timeout = %s, expected %s",
			client.httpClient.Timeout,
			expectedTimeout,
		)
	}
}

func TestNewClientRejectsInvalidConfiguration(t *testing.T) {
	testCases := []struct {
		name      string
		options   ClientOptions
		errorText string
	}{
		{
			name: "missing base URL",
			options: ClientOptions{
				TokenID:     "sync@pve!adguard-sync",
				TokenSecret: "test-secret",
			},
			errorText: "base URL must not be empty",
		},
		{
			name: "invalid base URL scheme",
			options: ClientOptions{
				BaseURL:     "ftp://proxmox.example.com",
				TokenID:     "sync@pve!adguard-sync",
				TokenSecret: "test-secret",
			},
			errorText: "scheme must be http or https",
		},
		{
			name: "missing token ID",
			options: ClientOptions{
				BaseURL:     "https://proxmox.example.com",
				TokenSecret: "test-secret",
			},
			errorText: "token ID must not be empty",
		},
		{
			name: "missing token secret",
			options: ClientOptions{
				BaseURL: "https://proxmox.example.com",
				TokenID: "sync@pve!adguard-sync",
			},
			errorText: "token secret must not be empty",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewClient(testCase.options)
			if err == nil {
				t.Fatal(
					"NewClient() returned nil error, expected validation error",
				)
			}

			if !strings.Contains(err.Error(), testCase.errorText) {
				t.Errorf(
					"error = %q, expected it to contain %q",
					err,
					testCase.errorText,
				)
			}
		})
	}
}
