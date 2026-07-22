package proxmox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetQEMUAgentInterfaces(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			expectedPath :=
				"/api2/json/nodes/pm/qemu/101/" +
					"agent/network-get-interfaces"

			if request.URL.Path != expectedPath {
				t.Errorf(
					"request path = %q, expected %q",
					request.URL.Path,
					expectedPath,
				)
			}

			writer.Header().Set(
				"Content-Type",
				"application/json",
			)

			_, _ = writer.Write([]byte(`{
				"data": {
					"result": [
						{
							"name": "lo",
							"hardware-address": "00:00:00:00:00:00",
							"ip-addresses": [
								{
									"ip-address-type": "ipv4",
									"ip-address": "127.0.0.1",
									"prefix": 8
								}
							]
						},
						{
							"name": "ens18",
							"hardware-address": "BC:24:11:AA:BB:CC",
							"ip-addresses": [
								{
									"ip-address-type": "ipv4",
									"ip-address": "172.20.20.10",
									"prefix": 24
								}
							]
						}
					]
				}
			}`))
		}),
	)
	defer server.Close()

	client, err := NewClient(ClientOptions{
		BaseURL:     server.URL + "/api2/json",
		TokenID:     "dns-sync@pve!adguard-sync",
		TokenSecret: "test-secret",
		VerifyTLS:   true,
	})
	if err != nil {
		t.Fatalf(
			"NewClient() returned an unexpected error: %v",
			err,
		)
	}

	interfaces, err := client.GetQEMUAgentInterfaces(
		context.Background(),
		"pm",
		101,
	)
	if err != nil {
		t.Fatalf(
			"GetQEMUAgentInterfaces() returned an unexpected error: %v",
			err,
		)
	}

	if len(interfaces) != 2 {
		t.Fatalf(
			"len(interfaces) = %d, expected 2",
			len(interfaces),
		)
	}

	if interfaces[1].Name != "ens18" {
		t.Errorf(
			"interfaces[1].Name = %q, expected %q",
			interfaces[1].Name,
			"ens18",
		)
	}

	if len(interfaces[1].IPAddresses) != 1 {
		t.Fatalf(
			"len(interfaces[1].IPAddresses) = %d, expected 1",
			len(interfaces[1].IPAddresses),
		)
	}

	if interfaces[1].IPAddresses[0].Address !=
		"172.20.20.10" {
		t.Errorf(
			"address = %q, expected %q",
			interfaces[1].IPAddresses[0].Address,
			"172.20.20.10",
		)
	}
}

func TestGetQEMUAgentInterfacesRejectsInvalidIdentity(
	t *testing.T,
) {
	client, err := NewClient(ClientOptions{
		BaseURL:     "https://proxmox.example.com/api2/json",
		TokenID:     "dns-sync@pve!adguard-sync",
		TokenSecret: "test-secret",
		VerifyTLS:   true,
	})
	if err != nil {
		t.Fatalf(
			"NewClient() returned an unexpected error: %v",
			err,
		)
	}

	testCases := []struct {
		name      string
		node      string
		vmid      int
		errorText string
	}{
		{
			name:      "empty node",
			node:      "",
			vmid:      101,
			errorText: "node must not be empty",
		},
		{
			name:      "invalid VMID",
			node:      "pm",
			vmid:      0,
			errorText: "VMID must be greater than zero",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := client.GetQEMUAgentInterfaces(
				context.Background(),
				testCase.node,
				testCase.vmid,
			)

			if err == nil {
				t.Fatal(
					"GetQEMUAgentInterfaces() returned nil error",
				)
			}

			if !strings.Contains(
				err.Error(),
				testCase.errorText,
			) {
				t.Errorf(
					"error = %q, expected to contain %q",
					err,
					testCase.errorText,
				)
			}
		})
	}
}
