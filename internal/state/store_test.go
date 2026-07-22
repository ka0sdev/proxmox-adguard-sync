package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingStateReturnsEmpty(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"missing",
		"state.json",
	)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() returned an unexpected error: %v", err)
	}

	stateFile, err := store.Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if stateFile.Version != CurrentVersion {
		t.Errorf(
			"Version = %d, expected %d",
			stateFile.Version,
			CurrentVersion,
		)
	}

	if len(stateFile.Records) != 0 {
		t.Errorf(
			"len(Records) = %d, expected 0",
			len(stateFile.Records),
		)
	}
}

func TestSaveAndLoadState(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"data",
		"state.json",
	)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() returned an unexpected error: %v", err)
	}

	err = store.Save(File{
		Records: []Record{
			{
				Domain: "LXC-DNS.Internal.",
				Answer: "172.20.0.4",
			},
			{
				Domain: "lxc-proxy.internal",
				Answer: "172.20.0.8",
			},
		},
	})
	if err != nil {
		t.Fatalf("Save() returned an unexpected error: %v", err)
	}

	stateFile, err := store.Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if len(stateFile.Records) != 2 {
		t.Fatalf(
			"len(Records) = %d, expected 2",
			len(stateFile.Records),
		)
	}

	if stateFile.Records[0].Domain != "lxc-dns.internal" {
		t.Errorf(
			"Records[0].Domain = %q",
			stateFile.Records[0].Domain,
		)
	}

	if stateFile.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() returned an unexpected error: %v", err)
	}

	if permissions := fileInfo.Mode().Perm(); permissions != 0o600 {
		t.Errorf(
			"permissions = %o, expected 600",
			permissions,
		)
	}
}

func TestLoadRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	if err := os.WriteFile(
		path,
		[]byte(`{"version":`),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile() returned an unexpected error: %v", err)
	}

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() returned an unexpected error: %v", err)
	}

	_, err = store.Load()
	if err == nil {
		t.Fatal("Load() returned nil error")
	}

	if !strings.Contains(err.Error(), "decode state file") {
		t.Errorf(
			"error = %q, expected decoding error",
			err,
		)
	}
}

func TestLoadRejectsUnsupportedVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	if err := os.WriteFile(
		path,
		[]byte(`{"version":999,"records":[]}`),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile() returned an unexpected error: %v", err)
	}

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() returned an unexpected error: %v", err)
	}

	_, err = store.Load()
	if err == nil {
		t.Fatal("Load() returned nil error")
	}

	if !strings.Contains(
		err.Error(),
		"unsupported state file version",
	) {
		t.Errorf(
			"error = %q, expected version error",
			err,
		)
	}
}

func TestManagedRecords(t *testing.T) {
	stateFile := File{
		Records: []Record{
			{
				Domain: "One.Internal.",
				Answer: "172.20.0.1",
			},
			{
				Domain: "two.internal",
				Answer: "172.20.0.2",
			},
		},
	}

	managed := stateFile.ManagedRecords()

	if managed["one.internal"] != "172.20.0.1" {
		t.Errorf(
			"managed one.internal = %q",
			managed["one.internal"],
		)
	}

	if managed["two.internal"] != "172.20.0.2" {
		t.Errorf(
			"managed two.internal = %q",
			managed["two.internal"],
		)
	}
}
