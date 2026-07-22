package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const CurrentVersion = 1

type File struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	Records   []Record  `json:"records"`
}

type Record struct {
	Domain string `json:"domain"`
	Answer string `json:"answer"`
}

type Store struct {
	path string
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("state file path must not be empty")
	}

	return &Store{
		path: filepath.Clean(path),
	}, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() (File, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Empty(), nil
		}

		return File{}, fmt.Errorf(
			"read state file %q: %w",
			s.path,
			err,
		)
	}

	var stateFile File

	if err := json.Unmarshal(content, &stateFile); err != nil {
		return File{}, fmt.Errorf(
			"decode state file %q: %w",
			s.path,
			err,
		)
	}

	if stateFile.Version != CurrentVersion {
		return File{}, fmt.Errorf(
			"unsupported state file version %d",
			stateFile.Version,
		)
	}

	stateFile.Records = normalizeRecords(stateFile.Records)

	return stateFile, nil
}

func (s *Store) Save(stateFile File) error {
	stateFile.Version = CurrentVersion
	stateFile.UpdatedAt = time.Now().UTC()
	stateFile.Records = normalizeRecords(stateFile.Records)

	parentDirectory := filepath.Dir(s.path)

	if err := os.MkdirAll(parentDirectory, 0o750); err != nil {
		return fmt.Errorf(
			"create state directory %q: %w",
			parentDirectory,
			err,
		)
	}

	content, err := json.MarshalIndent(stateFile, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state file: %w", err)
	}

	content = append(content, '\n')

	temporaryFile, err := os.CreateTemp(
		parentDirectory,
		".proxmox-adguard-sync-state-*",
	)
	if err != nil {
		return fmt.Errorf(
			"create temporary state file: %w",
			err,
		)
	}

	temporaryPath := temporaryFile.Name()
	cleanup := true

	defer func() {
		_ = temporaryFile.Close()

		if cleanup {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := temporaryFile.Chmod(0o600); err != nil {
		return fmt.Errorf(
			"set temporary state file permissions: %w",
			err,
		)
	}

	if _, err := temporaryFile.Write(content); err != nil {
		return fmt.Errorf(
			"write temporary state file: %w",
			err,
		)
	}

	if err := temporaryFile.Sync(); err != nil {
		return fmt.Errorf(
			"sync temporary state file: %w",
			err,
		)
	}

	if err := temporaryFile.Close(); err != nil {
		return fmt.Errorf(
			"close temporary state file: %w",
			err,
		)
	}

	if err := os.Rename(temporaryPath, s.path); err != nil {
		return fmt.Errorf(
			"replace state file %q: %w",
			s.path,
			err,
		)
	}

	cleanup = false

	return nil
}

func Empty() File {
	return File{
		Version: CurrentVersion,
		Records: []Record{},
	}
}

func (f File) ManagedRecords() map[string]string {
	records := make(map[string]string, len(f.Records))

	for _, record := range f.Records {
		domain := normalizeDomain(record.Domain)
		if domain == "" {
			continue
		}

		records[domain] = strings.TrimSpace(record.Answer)
	}

	return records
}

func normalizeRecords(records []Record) []Record {
	byDomain := make(map[string]Record)

	for _, record := range records {
		domain := normalizeDomain(record.Domain)
		answer := strings.TrimSpace(record.Answer)

		if domain == "" || answer == "" {
			continue
		}

		byDomain[domain] = Record{
			Domain: domain,
			Answer: answer,
		}
	}

	normalized := make([]Record, 0, len(byDomain))

	for _, record := range byDomain {
		normalized = append(normalized, record)
	}

	sort.Slice(
		normalized,
		func(first, second int) bool {
			return normalized[first].Domain <
				normalized[second].Domain
		},
	)

	return normalized
}

func normalizeDomain(value string) string {
	return strings.ToLower(
		strings.Trim(
			strings.TrimSpace(value),
			".",
		),
	)
}
