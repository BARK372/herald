package project

// Project represents a configured project that Herald can operate on.
type Project struct {
	Name               string
	Path               string
	Description        string
	Default            bool
	AllowedTools       []string
	MaxConcurrentTasks int
	Git                GitConfig
}

type GitConfig struct {
	AutoBranch   bool
	AutoStash    bool
	AutoCommit   bool
	BranchPrefix string
}
