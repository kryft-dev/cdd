package git

import (
	"fmt"
	"net/url"
	"strings"
)

// WebURL rewrites a git remote URL to the home page of its repository: https
// (http stays http), without the ".git" suffix, the port or user ssh used,
// or any credentials. It accepts the scp-style "git@host:owner/repo.git"
// and the ssh, git, http and https URL forms. A local path or file URL has
// no home page and is an error.
func WebURL(remote string) (string, error) {
	raw := strings.TrimSpace(remote)
	errNotHosted := fmt.Errorf("remote %q is not a hosted repository", raw)

	if !strings.Contains(raw, "://") {
		host, path, ok := strings.Cut(raw, ":")
		if !ok || host == "" || strings.Contains(host, "/") {
			return "", errNotHosted
		}
		if _, h, found := strings.Cut(host, "@"); found {
			host = h
		}
		return web("https", host, path)
	}

	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return "", errNotHosted
	}
	switch u.Scheme {
	case "http":
		return web("http", u.Host, u.Path)
	case "https":
		return web("https", u.Host, u.Path)
	case "ssh", "git":
		// The port is the ssh daemon's, not the web server's.
		return web("https", u.Hostname(), u.Path)
	}
	return "", errNotHosted
}

// web joins the parts of a home page URL, dropping the path's slashes at
// both ends and its ".git" suffix.
func web(scheme, host, path string) (string, error) {
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	if path == "" {
		return "", fmt.Errorf("remote on %s names no repository", host)
	}
	return scheme + "://" + host + "/" + path, nil
}
