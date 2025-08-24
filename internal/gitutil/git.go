package gitutil

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func InferRepoName(repoRoot string) string {
	cmd := exec.Command("git", "-C", repoRoot, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return filepath.Base(repoRoot)
	}
	url := strings.TrimSpace(string(out))
	re := regexp.MustCompile(`[:/](?P<owner>[^/]+)/(?P<repo>[^/]+?)(?:\.git)?$`)
	m := re.FindStringSubmatch(url)
	if len(m) == 0 {
		return filepath.Base(repoRoot)
	}
	// fmt.Println("InferRepoName", m[1]+"/"+m[2])
	return m[1] + "/" + m[2]
}

func ResolveCommit(repoRoot, commitRef string) string {
	if commitRef != "" {
		// try to resolve
		cmd := exec.Command("git", "-C", repoRoot, "rev-parse", commitRef)
		if b, err := cmd.Output(); err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	cmd := exec.Command("git", "-C", repoRoot, "rev-parse", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "unknown"
	}
	return strings.TrimSpace(out.String())
}
