package proxmox

import (
	"encoding/json"
	"testing"
)

func TestGuestIsRunning(t *testing.T) {
	testCases := []struct {
		name     string
		status   string
		expected bool
	}{
		{
			name:     "running guest",
			status:   "running",
			expected: true,
		},
		{
			name:     "stopped guest",
			status:   "stopped",
			expected: false,
		},
		{
			name:     "empty status",
			status:   "",
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			guest := Guest{
				Status: testCase.status,
			}

			actual := guest.IsRunning()

			if actual != testCase.expected {
				t.Errorf(
					"IsRunning() = %t, expected %t",
					actual,
					testCase.expected,
				)
			}
		})
	}
}

func TestGuestParsedTags(t *testing.T) {
	guest := Guest{
		Tags: "dns; infrastructure,lxc;; network ",
	}

	actual := guest.ParsedTags()

	expected := []string{
		"dns",
		"infrastructure",
		"lxc",
		"network",
	}

	if len(actual) != len(expected) {
		t.Fatalf(
			"len(ParsedTags()) = %d, expected %d",
			len(actual),
			len(expected),
		)
	}

	for index := range expected {
		if actual[index] != expected[index] {
			t.Errorf(
				"ParsedTags()[%d] = %q, expected %q",
				index,
				actual[index],
				expected[index],
			)
		}
	}
}

func TestLXCConfigStringValue(t *testing.T) {
	config := LXCConfig{
		"net0": json.RawMessage(
			`"name=eth0,ip=172.20.0.4/16"`,
		),
		"memory": json.RawMessage(`2048`),
	}

	if actual := config.StringValue("net0"); actual !=
		"name=eth0,ip=172.20.0.4/16" {
		t.Errorf(
			"StringValue(net0) = %q",
			actual,
		)
	}

	if actual := config.StringValue("memory"); actual != "" {
		t.Errorf(
			"StringValue(memory) = %q, expected empty string",
			actual,
		)
	}
}
