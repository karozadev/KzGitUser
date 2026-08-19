// Package scan discovers Git repositories under a directory tree.
package scan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// DefaultMaxWorkers bounds directory-read concurrency when no explicit
// limit is given to Repos.
const DefaultMaxWorkers = 16

// Repos concurrently walks root looking for Git repository roots — any
// directory containing a ".git" entry (a directory for a normal clone, or
// a file for a worktree/submodule). It never descends into a directory
// once identified as a repository, skips hidden directories (dot-prefixed,
// e.g. ".cache", ".vscode"), and silently ignores directories it can't
// read (permission errors, broken mounts, etc.) rather than failing the
// whole scan.
//
// Symlinked directories are skipped implicitly: os.DirEntry reports a
// symlink's own type (ModeSymlink), not the type of what it points to, so
// IsDir() is false for them — which also means Repos never risks an
// infinite loop through a symlink cycle.
func Repos(root string, maxWorkers int) []string {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers
	}

	var (
		mu    sync.Mutex
		repos []string
		wg    sync.WaitGroup
	)
	sem := make(chan struct{}, maxWorkers)

	var walk func(dir string)
	walk = func(dir string) {
		defer wg.Done()

		sem <- struct{}{}
		entries, err := os.ReadDir(dir)
		<-sem
		if err != nil {
			return
		}

		for _, e := range entries {
			if e.Name() == ".git" {
				mu.Lock()
				repos = append(repos, dir)
				mu.Unlock()
				return
			}
		}

		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			wg.Add(1)
			go walk(filepath.Join(dir, e.Name()))
		}
	}

	wg.Add(1)
	go walk(root)
	wg.Wait()

	sort.Strings(repos)
	return repos
}
