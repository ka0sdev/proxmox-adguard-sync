package proxmox

type Version struct {
	Version string `json:"version"`
	Release string `json:"release"`
	RepoID  string `json:"repoid"`
}

type apiResponse[T any] struct {
	Data T `json:"data"`
}
