package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func (s *Store) BackupPath() string {
	return s.path + ".bak"
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

	stateFile, err := decodeStateFile(
		content,
		s.path,
	)
	if err != nil {
		return File{}, err
	}

	return stateFile, nil
}

func (s *Store) Save(stateFile File) error {
	stateFile.Version = CurrentVersion
	stateFile.UpdatedAt = time.Now().UTC()

	if err := validateRecords(
		stateFile.Records,
	); err != nil {
		return fmt.Errorf(
			"validate state file: %w",
			err,
		)
	}

	stateFile.Records = normalizeRecords(
		stateFile.Records,
	)

	parentDirectory := filepath.Dir(s.path)

	if err := os.MkdirAll(
		parentDirectory,
		0o750,
	); err != nil {
		return fmt.Errorf(
			"create state directory %q: %w",
			parentDirectory,
			err,
		)
	}

	content, err := json.MarshalIndent(
		stateFile,
		"",
		"  ",
	)
	if err != nil {
		return fmt.Errorf(
			"encode state file: %w",
			err,
		)
	}

	content = append(content, '\n')

	existingContent, err := os.ReadFile(s.path)
	switch {
	case err == nil:
		if _, decodeErr := decodeStateFile(
			existingContent,
			s.path,
		); decodeErr != nil {
			return fmt.Errorf(
				"refusing to replace invalid existing state file: %w",
				decodeErr,
			)
		}

		if err := writeAtomicFile(
			s.BackupPath(),
			existingContent,
		); err != nil {
			return fmt.Errorf(
				"write state backup %q: %w",
				s.BackupPath(),
				err,
			)
		}

	case errors.Is(err, os.ErrNotExist):
		// First save. There is no previous state to back up.

	default:
		return fmt.Errorf(
			"read existing state file %q before replacement: %w",
			s.path,
			err,
		)
	}

	if err := writeAtomicFile(
		s.path,
		content,
	); err != nil {
		return fmt.Errorf(
			"write state file %q: %w",
			s.path,
			err,
		)
	}

	return nil
}

func Empty() File {
	return File{
		Version: CurrentVersion,
		Records: []Record{},
	}
}

func (f File) ManagedRecords() map[string]string {
	records := make(
		map[string]string,
		len(f.Records),
	)

	for _, record := range f.Records {
		domain := normalizeDomain(
			record.Domain,
		)
		if domain == "" {
			continue
		}

		records[domain] = strings.TrimSpace(
			record.Answer,
		)
	}

	return records
}

func decodeStateFile(
	content []byte,
	path string,
) (File, error) {
	decoder := json.NewDecoder(
		bytes.NewReader(content),
	)

	decoder.DisallowUnknownFields()

	var stateFile File

	if err := decoder.Decode(
		&stateFile,
	); err != nil {
		return File{}, fmt.Errorf(
			"decode state file %q: %w",
			path,
			err,
		)
	}

	var trailing any

	err := decoder.Decode(&trailing)
	switch {
	case errors.Is(err, io.EOF):
		// Expected.

	case err == nil:
		return File{}, fmt.Errorf(
			"decode state file %q: unexpected trailing JSON data",
			path,
		)

	default:
		return File{}, fmt.Errorf(
			"decode state file %q: %w",
			path,
			err,
		)
	}

	if stateFile.Version != CurrentVersion {
		return File{}, fmt.Errorf(
			"unsupported state file version %d",
			stateFile.Version,
		)
	}

	if err := validateRecords(
		stateFile.Records,
	); err != nil {
		return File{}, fmt.Errorf(
			"validate state file %q: %w",
			path,
			err,
		)
	}

	stateFile.Records = normalizeRecords(
		stateFile.Records,
	)

	return stateFile, nil
}

func validateRecords(
	records []Record,
) error {
	seenDomains := make(
		map[string]struct{},
		len(records),
	)

	for index, record := range records {
		domain := normalizeDomain(
			record.Domain,
		)

		answer := strings.TrimSpace(
			record.Answer,
		)

		if domain == "" {
			return fmt.Errorf(
				"record %d has an empty domain",
				index,
			)
		}

		if answer == "" {
			return fmt.Errorf(
				"record %d for domain %q has an empty answer",
				index,
				domain,
			)
		}

		if _, exists := seenDomains[domain]; exists {
			return fmt.Errorf(
				"duplicate managed domain %q",
				domain,
			)
		}

		seenDomains[domain] = struct{}{}
	}

	return nil
}

func writeAtomicFile(
	path string,
	content []byte,
) error {
	parentDirectory := filepath.Dir(path)

	temporaryFile, err := os.CreateTemp(
		parentDirectory,
		".proxmox-adguard-sync-state-*",
	)
	if err != nil {
		return fmt.Errorf(
			"create temporary file: %w",
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

	if err := temporaryFile.Chmod(
		0o600,
	); err != nil {
		return fmt.Errorf(
			"set temporary file permissions: %w",
			err,
		)
	}

	if _, err := temporaryFile.Write(
		content,
	); err != nil {
		return fmt.Errorf(
			"write temporary file: %w",
			err,
		)
	}

	if err := temporaryFile.Sync(); err != nil {
		return fmt.Errorf(
			"sync temporary file: %w",
			err,
		)
	}

	if err := temporaryFile.Close(); err != nil {
		return fmt.Errorf(
			"close temporary file: %w",
			err,
		)
	}

	if err := os.Rename(
		temporaryPath,
		path,
	); err != nil {
		return fmt.Errorf(
			"replace file: %w",
			err,
		)
	}

	cleanup = false

	return nil
}

func normalizeRecords(
	records []Record,
) []Record {
	normalized := make(
		[]Record,
		0,
		len(records),
	)

	for _, record := range records {
		normalized = append(
			normalized,
			Record{
				Domain: normalizeDomain(
					record.Domain,
				),
				Answer: strings.TrimSpace(
					record.Answer,
				),
			},
		)
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
