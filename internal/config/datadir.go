package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// errNoHome is returned when the home directory is needed but unknown.
var errNoHome = errors.New("cannot find the home directory (set CINEDECK_DATA_DIR)")

// DataDir returns the directory for cinedeck's local data (the library database).
func DataDir() (string, error) {
	home, _ := os.UserHomeDir()
	return dataDirFor(runtime.GOOS, os.Getenv, home)
}

// LibraryPath returns the library database file: <DataDir()>/library.db.
func LibraryPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "library.db"), nil
}

// dataDirFor is the pure rule behind DataDir. getenv reads an environment
// variable; home is the user's home directory ("" when unknown).
func dataDirFor(goos string, getenv func(string) string, home string) (string, error) {
	if v := getenv("CINEDECK_DATA_DIR"); v != "" {
		return v, nil
	}

	switch goos {
	case "windows":
		if v := getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "cinedeck"), nil
		}
		if home == "" {
			return "", errNoHome
		}
		return filepath.Join(home, "AppData", "Local", "cinedeck"), nil
	case "darwin":
		if home == "" {
			return "", errNoHome
		}
		return filepath.Join(home, "Library", "Application Support", "cinedeck"), nil
	default:
		if v := getenv("XDG_DATA_HOME"); v != "" && strings.HasPrefix(v, "/") {
			return filepath.Join(v, "cinedeck"), nil
		}
		if home == "" {
			return "", errNoHome
		}
		return filepath.Join(home, ".local", "share", "cinedeck"), nil
	}
}
