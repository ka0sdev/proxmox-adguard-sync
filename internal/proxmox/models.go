package proxmox

import "strings"

type Version struct {
	Version string `json:"version"`
	Release string `json:"release"`
	RepoID  string `json:"repoid"`
}

type GuestType string

const (
	GuestTypeQEMU GuestType = "qemu"
	GuestTypeLXC  GuestType = "lxc"
)

type Guest struct {
	VMID     int       `json:"vmid"`
	Name     string    `json:"name"`
	Node     string    `json:"node"`
	Type     GuestType `json:"type"`
	Status   string    `json:"status"`
	Tags     string    `json:"tags"`
	Template int       `json:"template"`
}

func (g Guest) IsRunning() bool {
	return g.Status == "running"
}

func (g Guest) IsTemplate() bool {
	return g.Template == 1
}

func (g Guest) ParsedTags() []string {
	if strings.TrimSpace(g.Tags) == "" {
		return nil
	}

	parts := strings.FieldsFunc(
		g.Tags,
		func(character rune) bool {
			return character == ';' || character == ','
		},
	)

	tags := make([]string, 0, len(parts))

	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag != "" {
			tags = append(tags, tag)
		}
	}

	return tags
}

type apiResponse[T any] struct {
	Data T `json:"data"`
}
