package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

// Ops provides Git operations for task management.
type Ops struct {
	repoPath string
}

// NewOps creates a Git operations helper for the given repository path.
func NewOps(repoPath string) *Ops {
	return &Ops{repoPath: repoPath}
}

// CurrentBranch returns the current branch name.
// Uses symbolic-ref which works on unborn branches (no commits yet).
func (g *Ops) CurrentBranch(ctx context.Context) (string, error) {
	out, err := g.run(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("getting current branch: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// HasCommits returns true if the repository has at least one commit.
func (g *Ops) HasCommits(ctx context.Context) bool {
	_, err := g.run(ctx, "rev-parse", "HEAD")
	return err == nil
}

// IsClean returns true if the working tree has no uncommitted changes.
func (g *Ops) IsClean(ctx context.Context) (bool, error) {
	out, err := g.run(ctx, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("checking git status: %w", err)
	}
	return strings.TrimSpace(out) == "", nil
}

// Diff returns the diff between two refs. If toRef is empty, diffs against HEAD.
func (g *Ops) Diff(ctx context.Context, fromRef, toRef string) (string, error) {
	args := []string{"diff", "--stat", "--patch"}
	if toRef != "" {
		args = append(args, fromRef+"..."+toRef)
	} else {
		args = append(args, fromRef)
	}

	out, err := g.run(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("getting diff: %w", err)
	}
	return out, nil
}

// DiffStat returns a summary of changes (files changed, insertions, deletions).
func (g *Ops) DiffStat(ctx context.Context, fromRef, toRef string) (string, error) {
	args := []string{"diff", "--stat"}
	if toRef != "" {
		args = append(args, fromRef+"..."+toRef)
	} else {
		args = append(args, fromRef)
	}

	out, err := g.run(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("getting diff stat: %w", err)
	}
	return out, nil
}

// CreateBranch creates and checks out a new branch from the current HEAD.
func (g *Ops) CreateBranch(ctx context.Context, name string) error {
	if _, err := g.run(ctx, "checkout", "-b", name); err != nil {
		return fmt.Errorf("creating branch %q: %w", name, err)
	}
	return nil
}

// Checkout switches to the given branch.
func (g *Ops) Checkout(ctx context.Context, branch string) error {
	if _, err := g.run(ctx, "checkout", branch); err != nil {
		return fmt.Errorf("checking out %q: %w", branch, err)
	}
	return nil
}

// Stash saves uncommitted changes to the stash.
func (g *Ops) Stash(ctx context.Context) error {
	if _, err := g.run(ctx, "stash", "push", "-m", "herald: auto-stash before task"); err != nil {
		return fmt.Errorf("stashing changes: %w", err)
	}
	return nil
}

// StashPop restores the most recent stash.
func (g *Ops) StashPop(ctx context.Context) error {
	if _, err := g.run(ctx, "stash", "pop"); err != nil {
		return fmt.Errorf("popping stash: %w", err)
	}
	return nil
}

// IsGitRepo returns true if the path is a git repository.
func (g *Ops) IsGitRepo(ctx context.Context) bool {
	_, err := g.run(ctx, "rev-parse", "--git-dir")
	return err == nil
}

// Log returns the last N commit messages.
func (g *Ops) Log(ctx context.Context, n int) (string, error) {
	out, err := g.run(ctx, "log", "--oneline", fmt.Sprintf("-n%d", n))
	if err != nil {
		return "", fmt.Errorf("getting log: %w", err)
	}
	return out, nil
}

func (g *Ops) run(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %s", err, strings.TrimSpace(stderr.String()))
	}

	return stdout.String(), nil
}
