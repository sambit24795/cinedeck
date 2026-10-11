package catalog

import (
	"fmt"
	"strconv"
	"strings"
)

// MediaType is the kind of catalog entry an ID refers to.
type MediaType string

const (
	Movie MediaType = "movie"
	TV    MediaType = "tv"
)

// ID identifies one movie or TV show.
type ID struct {
	Type MediaType
	N    int64
}

// String renders the ID as "type:n", for example "movie:27205".
func (id ID) String() string {
	return string(id.Type) + ":" + strconv.FormatInt(id.N, 10)
}

// MarshalText makes ID encode as a JSON string.
func (id ID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

// UnmarshalText parses the JSON string form of an ID.
func (id *ID) UnmarshalText(text []byte) error {
	parsed, err := ParseID(string(text))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

// ParseID parses a string of the form "type:n".
func ParseID(s string) (ID, error) {
	typ, num, ok := strings.Cut(s, ":")
	if !ok {
		return ID{}, fmt.Errorf("invalid id %q", s)
	}

	var mt MediaType
	switch typ {
	case string(Movie):
		mt = Movie
	case string(TV):
		mt = TV
	default:
		return ID{}, fmt.Errorf("invalid id %q", s)
	}

	n, err := strconv.ParseInt(num, 10, 64)
	if err != nil || n < 0 {
		return ID{}, fmt.Errorf("invalid id %q", s)
	}
	return ID{Type: mt, N: n}, nil
}
