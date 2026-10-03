package common

import (
	"errors"
	"fmt"
	"net/url"
	"path"
)

var (
	ErrUnsupportedURLScheme = errors.New("unsupported URL scheme")
	ErrMissingURLHost       = errors.New("missing URL host")
	ErrDomainNotAllowed     = errors.New("domain not allowed")
)

func ParseHttpURL(rawUrl string) (*url.URL, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedURLScheme, u.Scheme)
	}

	if u.Hostname() == "" {
		return nil, ErrMissingURLHost
	}

	return u, nil
}

func CheckDomains(rawUrl string, domains []string) error {
	u, err := ParseHttpURL(rawUrl)
	if err != nil {
		return err
	}

	if !matches(domains, u.Hostname()) {
		return fmt.Errorf("%w: %q", ErrDomainNotAllowed, u.Hostname())
	}

	return nil
}

func matches(domains []string, url string) bool {
	for _, d := range domains {
		ok, _ := path.Match(d, url)
		if ok {
			return true
		}
	}
	return false
}
