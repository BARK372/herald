package project

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/btouchard/herald/internal/config"
)

// Manager loads and manages configured projects.
type Manager struct {
	projects map[string]*Project
}

// NewManager creates a Manager from the config's project map.
func NewManager(projects map[string]config.Project) *Manager {
	m := &Manager{
		projects: make(map[string]*Project, len(projects)),
	}

	for name, cfg := range projects {
		p := &Project{
			Name:               name,
			Path:               cfg.Path,
			Description:        cfg.Description,
			Default:            cfg.Default,
			AllowedTools:       cfg.AllowedTools,
			MaxConcurrentTasks: cfg.MaxConcurrentTasks,
			Git: GitConfig{
				AutoBranch:   cfg.Git.AutoBranch,
				AutoStash:    cfg.Git.AutoStash,
				AutoCommit:   cfg.Git.AutoCommit,
				BranchPrefix: cfg.Git.BranchPrefix,
			},
		}
		if p.MaxConcurrentTasks < 1 {
			p.MaxConcurrentTasks = 1
		}
		if p.Git.BranchPrefix == "" {
			p.Git.BranchPrefix = "herald/"
		}
		m.projects[name] = p
	}

	return m
}

// Validate checks that all configured projects have valid paths.
func (m *Manager) Validate() error {
	for name, p := range m.projects {
		info, err := os.Stat(p.Path)
		if os.IsNotExist(err) {
			return fmt.Errorf("project %s: path %s does not exist", name, p.Path)
		}
		if err != nil {
			return fmt.Errorf("project %s: %w", name, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("project %s: path %s is not a directory", name, p.Path)
		}

		gitDir := filepath.Join(p.Path, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			slog.Warn("project is not a git repository", "project", name, "path", p.Path)
		}
	}
	return nil
}

// Get returns a project by name.
func (m *Manager) Get(name string) (*Project, error) {
	p, ok := m.projects[name]
	if !ok {
		return nil, fmt.Errorf("project %q not found", name)
	}
	return p, nil
}

// Default returns the default project.
func (m *Manager) Default() (*Project, error) {
	for _, p := range m.projects {
		if p.Default {
			return p, nil
		}
	}
	if len(m.projects) == 1 {
		for _, p := range m.projects {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no default project configured")
}

// Resolve returns a project by name, or the default if name is empty.
func (m *Manager) Resolve(name string) (*Project, error) {
	if name == "" {
		return m.Default()
	}
	return m.Get(name)
}

// All returns all configured projects.
func (m *Manager) All() []*Project {
	result := make([]*Project, 0, len(m.projects))
	for _, p := range m.projects {
		result = append(result, p)
	}
	return result
}

// GitBranch returns the current git branch for a project, or empty string.
// Uses symbolic-ref which works on unborn branches (no commits yet).
func (m *Manager) GitBranch(p *Project) string {
	out, err := exec.Command("git", "-C", p.Path, "symbolic-ref", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GitClean returns whether the project's working tree is clean.
func (m *Manager) GitClean(p *Project) bool {
	out, err := exec.Command("git", "-C", p.Path, "status", "--porcelain").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == ""
}
