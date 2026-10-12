package indexer

import (
	"errors"
	"net/url"
	"strings"
)

// ParseMagnet extracts info hash and title from a magnet link.
func ParseMagnet(magnet string) (infoHash, title string, err error) {
	magnet = strings.TrimSpace(magnet)
	if !strings.HasPrefix(magnet, "magnet:?") {
		return "", "", errors.Join(ErrParseFailed, errors.New("invalid magnet link"))
	}

	u, err := url.Parse(magnet)
	if err != nil {
		return "", "", errors.Join(ErrParseFailed, err)
	}

	query := u.Query()
	infoHash = strings.ToLower(query.Get("xt"))
	if infoHash == "" || !strings.HasPrefix(infoHash, "urn:btih:") {
		return "", "", errors.Join(ErrParseFailed, errors.New("missing or invalid xt parameter"))
	}
	infoHash = strings.TrimPrefix(infoHash, "urn:btih:")

	title = query.Get("dn")
	if title == "" {
		title = "unknown"
	}

	return infoHash, title, nil
}
