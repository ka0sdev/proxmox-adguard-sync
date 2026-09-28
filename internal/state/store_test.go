package state

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadMissingStateReturnsEmpty(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"missing",
		"state.json",
	)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	stateFile, err := store.Load()
	if err != nil {
		t.Fatalf(
			"Load() returned an unexpected error: %v",
			err,
		)
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

func TestSaveAndLoadState(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"data",
		"state.json",
	)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	err = store.Save(
		File{
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
		},
	)
	if err != nil {
		t.Fatalf(
			"Save() returned an unexpected error: %v",
			err,
		)
	}

	stateFile, err := store.Load()
	if err != nil {
		t.Fatalf(
			"Load() returned an unexpected error: %v",
			err,
		)
	}

	if len(stateFile.Records) != 2 {
		t.Fatalf(
			"len(Records) = %d, expected 2",
			len(stateFile.Records),
		)
	}

	if stateFile.Records[0].Domain !=
		"lxc-dns.internal" {
		t.Errorf(
			"Records[0].Domain = %q",
			stateFile.Records[0].Domain,
		)
	}

	if stateFile.UpdatedAt.IsZero() {
		t.Error(
			"UpdatedAt is zero",
		)
	}

	if runtime.GOOS != "windows" {
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf(
				"Stat() returned an unexpected error: %v",
				err,
			)
		}

		if permissions := fileInfo.Mode().Perm(); permissions != 0o600 {
			t.Errorf(
				"permissions = %o, expected 600",
				permissions,
			)
		}
	}
}

func TestFirstSaveDoesNotCreateBackup(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	err = store.Save(
		File{
			Records: []Record{
				{
					Domain: "one.internal",
					Answer: "172.20.0.1",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf(
			"Save() returned an unexpected error: %v",
			err,
		)
	}

	_, err = os.Stat(
		store.BackupPath(),
	)

	if !errors.Is(
		err,
		os.ErrNotExist,
	) {
		t.Fatalf(
			"backup exists after first save; Stat() error = %v",
			err,
		)
	}
}

func TestSaveCreatesBackupOfPreviousState(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	err = store.Save(
		File{
			Records: []Record{
				{
					Domain: "one.internal",
					Answer: "172.20.0.1",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf(
			"first Save() returned an unexpected error: %v",
			err,
		)
	}

	firstContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(
			"ReadFile() returned an unexpected error: %v",
			err,
		)
	}

	err = store.Save(
		File{
			Records: []Record{
				{
					Domain: "one.internal",
					Answer: "172.20.0.2",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf(
			"second Save() returned an unexpected error: %v",
			err,
		)
	}

	backupContent, err := os.ReadFile(
		store.BackupPath(),
	)
	if err != nil {
		t.Fatalf(
			"read backup: %v",
			err,
		)
	}

	if string(backupContent) !=
		string(firstContent) {
		t.Error(
			"backup does not contain the previous state",
		)
	}

	if runtime.GOOS != "windows" {
		backupInfo, err := os.Stat(
			store.BackupPath(),
		)
		if err != nil {
			t.Fatalf(
				"stat backup: %v",
				err,
			)
		}

		if permissions := backupInfo.Mode().Perm(); permissions != 0o600 {
			t.Errorf(
				"backup permissions = %o, expected 600",
				permissions,
			)
		}
	}

	current, err := store.Load()
	if err != nil {
		t.Fatalf(
			"Load() returned an unexpected error: %v",
			err,
		)
	}

	if current.Records[0].Answer !=
		"172.20.0.2" {
		t.Errorf(
			"current answer = %q, expected 172.20.0.2",
			current.Records[0].Answer,
		)
	}
}

func TestLoadRejectsInvalidJSON(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	if err := os.WriteFile(
		path,
		[]byte(`{"version":`),
		0o600,
	); err != nil {
		t.Fatalf(
			"WriteFile() returned an unexpected error: %v",
			err,
		)
	}

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	_, err = store.Load()
	if err == nil {
		t.Fatal(
			"Load() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"decode state file",
	) {
		t.Errorf(
			"error = %q, expected decoding error",
			err,
		)
	}
}

func TestLoadRejectsUnsupportedVersion(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	if err := os.WriteFile(
		path,
		[]byte(
			`{"version":999,"records":[]}`,
		),
		0o600,
	); err != nil {
		t.Fatalf(
			"WriteFile() returned an unexpected error: %v",
			err,
		)
	}

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	_, err = store.Load()
	if err == nil {
		t.Fatal(
			"Load() returned nil error",
		)
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

func TestLoadRejectsUnknownFields(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	content := []byte(
		`{
  "version": 1,
  "updated_at": "2026-09-28T10:00:00Z",
  "records": [],
  "unexpected": true
}`,
	)

	if err := os.WriteFile(
		path,
		content,
		0o600,
	); err != nil {
		t.Fatalf(
			"WriteFile() returned an unexpected error: %v",
			err,
		)
	}

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	_, err = store.Load()
	if err == nil {
		t.Fatal(
			"Load() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"unknown field",
	) {
		t.Errorf(
			"error = %q, expected unknown field error",
			err,
		)
	}
}

func TestLoadRejectsTrailingJSON(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	content := []byte(
		`{"version":1,"records":[]} {"version":1,"records":[]}`,
	)

	if err := os.WriteFile(
		path,
		content,
		0o600,
	); err != nil {
		t.Fatalf(
			"WriteFile() returned an unexpected error: %v",
			err,
		)
	}

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	_, err = store.Load()
	if err == nil {
		t.Fatal(
			"Load() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"trailing JSON",
	) {
		t.Errorf(
			"error = %q, expected trailing JSON error",
			err,
		)
	}
}

func TestLoadRejectsEmptyDomain(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	content := []byte(
		`{
  "version": 1,
  "records": [
    {
      "domain": "",
      "answer": "172.20.0.4"
    }
  ]
}`,
	)

	if err := os.WriteFile(
		path,
		content,
		0o600,
	); err != nil {
		t.Fatalf(
			"WriteFile() returned an unexpected error: %v",
			err,
		)
	}

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	_, err = store.Load()
	if err == nil {
		t.Fatal(
			"Load() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"empty domain",
	) {
		t.Errorf(
			"error = %q, expected empty domain error",
			err,
		)
	}
}

func TestLoadRejectsEmptyAnswer(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	content := []byte(
		`{
  "version": 1,
  "records": [
    {
      "domain": "one.internal",
      "answer": ""
    }
  ]
}`,
	)

	if err := os.WriteFile(
		path,
		content,
		0o600,
	); err != nil {
		t.Fatalf(
			"WriteFile() returned an unexpected error: %v",
			err,
		)
	}

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	_, err = store.Load()
	if err == nil {
		t.Fatal(
			"Load() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"empty answer",
	) {
		t.Errorf(
			"error = %q, expected empty answer error",
			err,
		)
	}
}

func TestLoadRejectsDuplicateNormalizedDomain(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	content := []byte(
		`{
  "version": 1,
  "records": [
    {
      "domain": "DNS.INTERNAL",
      "answer": "172.20.0.4"
    },
    {
      "domain": "dns.internal.",
      "answer": "172.20.0.5"
    }
  ]
}`,
	)

	if err := os.WriteFile(
		path,
		content,
		0o600,
	); err != nil {
		t.Fatalf(
			"WriteFile() returned an unexpected error: %v",
			err,
		)
	}

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	_, err = store.Load()
	if err == nil {
		t.Fatal(
			"Load() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"duplicate managed domain",
	) {
		t.Errorf(
			"error = %q, expected duplicate domain error",
			err,
		)
	}
}

func TestSaveRejectsInvalidRecords(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	err = store.Save(
		File{
			Records: []Record{
				{
					Domain: "",
					Answer: "172.20.0.1",
				},
			},
		},
	)

	if err == nil {
		t.Fatal(
			"Save() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"empty domain",
	) {
		t.Errorf(
			"error = %q, expected empty domain error",
			err,
		)
	}

	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf(
			"state file exists after rejected save",
		)
	}
}

func TestSaveRejectsDuplicateDomains(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	err = store.Save(
		File{
			Records: []Record{
				{
					Domain: "one.internal",
					Answer: "172.20.0.1",
				},
				{
					Domain: "ONE.INTERNAL.",
					Answer: "172.20.0.2",
				},
			},
		},
	)

	if err == nil {
		t.Fatal(
			"Save() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"duplicate managed domain",
	) {
		t.Errorf(
			"error = %q, expected duplicate domain error",
			err,
		)
	}
}

func TestSaveRefusesToReplaceCorruptExistingState(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	corruptContent := []byte(
		`{"version":`,
	)

	if err := os.WriteFile(
		path,
		corruptContent,
		0o600,
	); err != nil {
		t.Fatalf(
			"WriteFile() returned an unexpected error: %v",
			err,
		)
	}

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf(
			"NewStore() returned an unexpected error: %v",
			err,
		)
	}

	err = store.Save(
		File{
			Records: []Record{
				{
					Domain: "one.internal",
					Answer: "172.20.0.1",
				},
			},
		},
	)

	if err == nil {
		t.Fatal(
			"Save() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"refusing to replace invalid existing state file",
	) {
		t.Errorf(
			"error = %q, expected replacement refusal",
			err,
		)
	}

	currentContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(
			"read existing state: %v",
			err,
		)
	}

	if string(currentContent) !=
		string(corruptContent) {
		t.Error(
			"corrupt state file was modified",
		)
	}

	if _, statErr := os.Stat(
		store.BackupPath(),
	); !errors.Is(
		statErr,
		os.ErrNotExist,
	) {
		t.Error(
			"backup was created from corrupt state",
		)
	}
}

func TestManagedRecords(
	t *testing.T,
) {
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

	if managed["one.internal"] !=
		"172.20.0.1" {
		t.Errorf(
			"managed one.internal = %q",
			managed["one.internal"],
		)
	}

	if managed["two.internal"] !=
		"172.20.0.2" {
		t.Errorf(
			"managed two.internal = %q",
			managed["two.internal"],
		)
	}
}
