package proxmox

import "testing"

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
