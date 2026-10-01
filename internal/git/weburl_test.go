package git_test

import (
	"testing"

	"github.com/kryft-dev/cdd/internal/git"
)

func TestWebURL(t *testing.T) {
	tests := []struct {
		name, remote, want string
	}{
		{"scp-style", "git@github.com:owner/repo.git", "https://github.com/owner/repo"},
		{"scp-style without .git", "git@github.com:owner/repo", "https://github.com/owner/repo"},
		{"scp-style without user", "github.com:owner/repo.git", "https://github.com/owner/repo"},
		{"scp-style subgroups", "git@gitlab.com:owner/group/sub/repo.git", "https://gitlab.com/owner/group/sub/repo"},
		{"ssh", "ssh://git@github.com/owner/repo.git", "https://github.com/owner/repo"},
		{"ssh with port", "ssh://git@host.example:2222/owner/repo.git", "https://host.example/owner/repo"},
		{"ssh subgroups", "ssh://git@gitlab.com/owner/group/repo.git", "https://gitlab.com/owner/group/repo"},
		{"git protocol", "git://github.com/owner/repo.git", "https://github.com/owner/repo"},
		{"https", "https://github.com/owner/repo.git", "https://github.com/owner/repo"},
		{"https without .git", "https://github.com/owner/repo", "https://github.com/owner/repo"},
		{"https with port", "https://git.example:8443/owner/repo.git", "https://git.example:8443/owner/repo"},
		{"https subgroups", "https://gitlab.com/owner/group/sub/repo.git", "https://gitlab.com/owner/group/sub/repo"},
		{"https with credentials", "https://user:token@github.com/owner/repo.git", "https://github.com/owner/repo"},
		{"http", "http://git.example/owner/repo.git", "http://git.example/owner/repo"},
		{"trailing slash", "https://github.com/owner/repo/", "https://github.com/owner/repo"},
		{"surrounding space", " git@github.com:owner/repo.git\n", "https://github.com/owner/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.WebURL(tt.remote)
			if err != nil || got != tt.want {
				t.Errorf("WebURL(%q) = %q, %v, want %q", tt.remote, got, err, tt.want)
			}
		})
	}
}

func TestWebURL_RejectsWhatIsNotAHostedRemote(t *testing.T) {
	for _, remote := range []string{"", "/srv/git/repo.git", "../repo", "file:///srv/repo.git", "repo.git"} {
		if got, err := git.WebURL(remote); err == nil {
			t.Errorf("WebURL(%q) = %q, want an error", remote, got)
		}
	}
}
