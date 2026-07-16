package proxmox

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
	VMID   int       `json:"vmid"`
	Name   string    `json:"name"`
	Node   string    `json:"node"`
	Type   GuestType `json:"type"`
	Status string    `json:"status"`
	Tags   string    `json:"tags"`
}

func (g Guest) IsRunning() bool {
	return g.Status == "running"
}

type apiResponse[T any] struct {
	Data T `json:"data"`
}
