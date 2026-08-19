// Package gitlog extracts per-author commit history from Git repositories,
// for building contribution statistics across one or many repos.
package gitlog

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// DefaultMaxWorkers bounds how many `git log` processes run concurrently
// when no explicit limit is given to Scan.
const DefaultMaxWorkers = 8

// logTimeout bounds a single repository's `git log` call, so one huge or
// pathological repo can't stall an entire scan.
const logTimeout = 30 * time.Second

// unitSeparator delimits the fields requested from `git log --format`; it's
// vanishingly unlikely to appear in an email or a date, unlike printable
// delimiters such as "|".
const unitSeparator = "\x1f"

// Entry is a single commit's repository, author email, and commit date
// (YYYY-MM-DD, in the local timezone `git log --date=short` uses).
type Entry struct {
	Repo  string
	Email string
	Date  string
}

// Scan runs `git log` across every given repository concurrently (bounded
// by maxWorkers) and returns every commit found since the given time,
// across all authors — deliberately unfiltered by author, so a single pass
// over each repo can serve any number of profiles (see CountsByEmail).
// Repositories that error out (not a valid Git repo, no commits, git
// missing, a timeout) are silently skipped so one bad repo can't abort the
// whole scan.
func Scan(repoPaths []string, since time.Time, maxWorkers int) []Entry {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers
	}

	var (
		mu      sync.Mutex
		entries []Entry
		wg      sync.WaitGroup
	)
	sem := make(chan struct{}, maxWorkers)

	for _, repo := range repoPaths {
		wg.Add(1)
		go func(repo string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			repoEntries, err := logRepo(repo, since)
			if err != nil {
				return
			}
			mu.Lock()
			entries = append(entries, repoEntries...)
			mu.Unlock()
		}(repo)
	}
	wg.Wait()
	return entries
}

func logRepo(repoPath string, since time.Time) ([]Entry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), logTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git",
		"log",
		"--since="+since.Format("2006-01-02"),
		"--format=%ae"+unitSeparator+"%ad",
		"--date=short",
	)
	cmd.Dir = repoPath

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var entries []Entry
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		email, date, ok := strings.Cut(line, unitSeparator)
		if !ok {
			continue
		}
		entries = append(entries, Entry{Repo: repoPath, Email: email, Date: date})
	}
	return entries, nil
}

// CountsByEmail buckets entries into a per-author (lowercased email) map of
// commit date to commit count, letting a single Scan serve any number of
// profiles without re-running `git log`.
func CountsByEmail(entries []Entry) map[string]map[string]int {
	byEmail := make(map[string]map[string]int)
	for _, e := range entries {
		email := strings.ToLower(e.Email)
		dates, ok := byEmail[email]
		if !ok {
			dates = make(map[string]int)
			byEmail[email] = dates
		}
		dates[e.Date]++
	}
	return byEmail
}
