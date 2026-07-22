package adguard

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListRewrites(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			if request.Method != http.MethodGet {
				t.Errorf(
					"method = %q, expected GET",
					request.Method,
				)
			}

			expectedPath := "/control/rewrite/list"

			if request.URL.Path != expectedPath {
				t.Errorf(
					"path = %q, expected %q",
					request.URL.Path,
					expectedPath,
				)
			}

			expectedAuthorization := "Basic " +
				base64.StdEncoding.EncodeToString(
					[]byte("admin:secret"),
				)

			if actual := request.Header.Get(
				"Authorization",
			); actual != expectedAuthorization {
				t.Errorf(
					"Authorization = %q, expected %q",
					actual,
					expectedAuthorization,
				)
			}

			writer.Header().Set(
				"Content-Type",
				"application/json",
			)

			_, _ = writer.Write([]byte(`[
				{
					"domain": "lxc-dns.internal",
					"answer": "172.20.0.4"
				},
				{
					"domain": "lxc-proxy-01.internal",
					"answer": "172.20.0.8"
				}
			]`))
		}),
	)
	defer server.Close()

	client, err := NewClient(ClientOptions{
		BaseURL:  server.URL,
		Username: "admin",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf(
			"NewClient() returned an unexpected error: %v",
			err,
		)
	}

	rewrites, err := client.ListRewrites(
		context.Background(),
	)
	if err != nil {
		t.Fatalf(
			"ListRewrites() returned an unexpected error: %v",
			err,
		)
	}

	if len(rewrites) != 2 {
		t.Fatalf(
			"len(rewrites) = %d, expected 2",
			len(rewrites),
		)
	}

	if rewrites[0].Domain != "lxc-dns.internal" {
		t.Errorf(
			"rewrites[0].Domain = %q",
			rewrites[0].Domain,
		)
	}

	if rewrites[0].Answer != "172.20.0.4" {
		t.Errorf(
			"rewrites[0].Answer = %q",
			rewrites[0].Answer,
		)
	}
}

func TestListRewritesSupportsControlBaseURL(
	t *testing.T,
) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			if request.URL.Path != "/control/rewrite/list" {
				t.Errorf(
					"path = %q, expected %q",
					request.URL.Path,
					"/control/rewrite/list",
				)
			}

			writer.Header().Set(
				"Content-Type",
				"application/json",
			)

			_, _ = writer.Write([]byte(`[]`))
		}),
	)
	defer server.Close()

	client, err := NewClient(ClientOptions{
		BaseURL:  server.URL + "/control",
		Username: "admin",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf(
			"NewClient() returned an unexpected error: %v",
			err,
		)
	}

	_, err = client.ListRewrites(context.Background())
	if err != nil {
		t.Fatalf(
			"ListRewrites() returned an unexpected error: %v",
			err,
		)
	}
}

func TestListRewritesReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			writer http.ResponseWriter,
			_ *http.Request,
		) {
			http.Error(
				writer,
				"Unauthorized",
				http.StatusUnauthorized,
			)
		}),
	)
	defer server.Close()

	client, err := NewClient(ClientOptions{
		BaseURL:  server.URL,
		Username: "admin",
		Password: "incorrect",
	})
	if err != nil {
		t.Fatalf(
			"NewClient() returned an unexpected error: %v",
			err,
		)
	}

	_, err = client.ListRewrites(context.Background())
	if err == nil {
		t.Fatal("ListRewrites() returned nil error")
	}

	if !strings.Contains(err.Error(), "HTTP 401") {
		t.Errorf(
			"error = %q, expected HTTP 401",
			err,
		)
	}
}

func TestNewClientRejectsInvalidOptions(t *testing.T) {
	testCases := []struct {
		name      string
		options   ClientOptions
		errorText string
	}{
		{
			name: "empty base URL",
			options: ClientOptions{
				Username: "admin",
				Password: "secret",
			},
			errorText: "base URL must not be empty",
		},
		{
			name: "unsupported scheme",
			options: ClientOptions{
				BaseURL:  "ftp://adguard.example.com",
				Username: "admin",
				Password: "secret",
			},
			errorText: "unsupported AdGuard URL scheme",
		},
		{
			name: "empty username",
			options: ClientOptions{
				BaseURL:  "http://adguard.example.com",
				Password: "secret",
			},
			errorText: "username must not be empty",
		},
		{
			name: "empty password",
			options: ClientOptions{
				BaseURL:  "http://adguard.example.com",
				Username: "admin",
			},
			errorText: "password must not be empty",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewClient(testCase.options)

			if err == nil {
				t.Fatal("NewClient() returned nil error")
			}

			if !strings.Contains(
				err.Error(),
				testCase.errorText,
			) {
				t.Errorf(
					"error = %q, expected %q",
					err,
					testCase.errorText,
				)
			}
		})
	}
}

func TestAddRewrite(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			if request.Method != http.MethodPost {
				t.Errorf(
					"method = %q, expected POST",
					request.Method,
				)
			}

			if request.URL.Path != "/control/rewrite/add" {
				t.Errorf(
					"path = %q",
					request.URL.Path,
				)
			}

			var payload Rewrite

			if err := json.NewDecoder(
				request.Body,
			).Decode(&payload); err != nil {
				t.Fatalf(
					"decode request: %v",
					err,
				)
			}

			if payload.Domain != "lxc-dns.internal" {
				t.Errorf(
					"Domain = %q",
					payload.Domain,
				)
			}

			if payload.Answer != "172.20.0.4" {
				t.Errorf(
					"Answer = %q",
					payload.Answer,
				)
			}

			writer.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	client, err := NewClient(ClientOptions{
		BaseURL:  server.URL,
		Username: "admin",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf(
			"NewClient() returned an unexpected error: %v",
			err,
		)
	}

	err = client.AddRewrite(
		context.Background(),
		Rewrite{
			Domain: "lxc-dns.internal",
			Answer: "172.20.0.4",
		},
	)
	if err != nil {
		t.Fatalf(
			"AddRewrite() returned an unexpected error: %v",
			err,
		)
	}
}

func TestUpdateRewrite(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			if request.Method != http.MethodPut {
				t.Errorf(
					"method = %q, expected PUT",
					request.Method,
				)
			}

			if request.URL.Path !=
				"/control/rewrite/update" {
				t.Errorf(
					"path = %q",
					request.URL.Path,
				)
			}

			var payload RewriteUpdate

			if err := json.NewDecoder(
				request.Body,
			).Decode(&payload); err != nil {
				t.Fatalf(
					"decode request: %v",
					err,
				)
			}

			if payload.Target.Answer !=
				"172.20.0.99" {
				t.Errorf(
					"Target.Answer = %q",
					payload.Target.Answer,
				)
			}

			if payload.Update.Answer !=
				"172.20.0.4" {
				t.Errorf(
					"Update.Answer = %q",
					payload.Update.Answer,
				)
			}

			writer.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	client, err := NewClient(ClientOptions{
		BaseURL:  server.URL,
		Username: "admin",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf(
			"NewClient() returned an unexpected error: %v",
			err,
		)
	}

	err = client.UpdateRewrite(
		context.Background(),
		Rewrite{
			Domain: "lxc-dns.internal",
			Answer: "172.20.0.99",
		},
		Rewrite{
			Domain: "lxc-dns.internal",
			Answer: "172.20.0.4",
		},
	)
	if err != nil {
		t.Fatalf(
			"UpdateRewrite() returned an unexpected error: %v",
			err,
		)
	}
}

func TestDeleteRewrite(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			if request.Method != http.MethodPost {
				t.Errorf(
					"method = %q, expected POST",
					request.Method,
				)
			}

			if request.URL.Path !=
				"/control/rewrite/delete" {
				t.Errorf(
					"path = %q",
					request.URL.Path,
				)
			}

			var payload Rewrite

			if err := json.NewDecoder(
				request.Body,
			).Decode(&payload); err != nil {
				t.Fatalf(
					"decode request: %v",
					err,
				)
			}

			if payload.Domain != "stale.internal" {
				t.Errorf(
					"Domain = %q",
					payload.Domain,
				)
			}

			writer.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	client, err := NewClient(ClientOptions{
		BaseURL:  server.URL,
		Username: "admin",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf(
			"NewClient() returned an unexpected error: %v",
			err,
		)
	}

	err = client.DeleteRewrite(
		context.Background(),
		Rewrite{
			Domain: "stale.internal",
			Answer: "172.20.0.50",
		},
	)
	if err != nil {
		t.Fatalf(
			"DeleteRewrite() returned an unexpected error: %v",
			err,
		)
	}
}
