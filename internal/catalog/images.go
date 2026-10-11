package catalog

const imageBaseURL = "https://image.tmdb.org/t/p/"

const (
	posterSize   = "w342"
	backdropSize = "w780"
	profileSize  = "w185"
)

// imageURL returns imageBaseURL + size + path, or "" when path is empty.
func imageURL(size, path string) string {
	if path == "" {
		return ""
	}
	return imageBaseURL + size + path
}
