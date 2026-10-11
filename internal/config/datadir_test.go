package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func fakeGetenv(env map[string]string) func(string) string {
	return func(key string) string {
		return env[key]
	}
}

func TestDataDirFor(t *testing.T) {
	tests := []struct {
		name string
		goos string
		env  map[string]string
		home string
		want string
	}{
		{
			name: "data dir wins on linux",
			goos: "linux",
			env:  map[string]string{"CINEDECK_DATA_DIR": "/x/y"},
			home: "/h",
			want: "/x/y",
		},
		{
			name: "data dir wins on windows",
			goos: "windows",
			env:  map[string]string{"CINEDECK_DATA_DIR": "/x/y"},
			home: "C:\\Users\\u",
			want: "/x/y",
		},
		{
			name: "data dir wins on darwin",
			goos: "darwin",
			env:  map[string]string{"CINEDECK_DATA_DIR": "/x/y"},
			home: "/Users/u",
			want: "/x/y",
		},
		{
			name: "data dir wins without home",
			goos: "linux",
			env:  map[string]string{"CINEDECK_DATA_DIR": "/x/y"},
			home: "",
			want: "/x/y",
		},
		{
			name: "linux xdg absolute",
			goos: "linux",
			env:  map[string]string{"XDG_DATA_HOME": "/xdg"},
			home: "/h",
			want: filepath.Join("/xdg", "cinedeck"),
		},
		{
			name: "linux xdg relative is ignored",
			goos: "linux",
			env:  map[string]string{"XDG_DATA_HOME": "relative"},
			home: "/h",
			want: filepath.Join("/h", ".local", "share", "cinedeck"),
		},
		{
			name: "linux default",
			goos: "linux",
			env:  map[string]string{},
			home: "/h",
			want: filepath.Join("/h", ".local", "share", "cinedeck"),
		},
		{
			name: "android default",
			goos: "android",
			env:  map[string]string{},
			home: "/h",
			want: filepath.Join("/h", ".local", "share", "cinedeck"),
		},
		{
			name: "darwin default",
			goos: "darwin",
			env:  map[string]string{},
			home: "/Users/u",
			want: filepath.Join("/Users/u", "Library", "Application Support", "cinedeck"),
		},
		{
			name: "windows localappdata",
			goos: "windows",
			env:  map[string]string{"LOCALAPPDATA": "C:\\L"},
			home: "C:\\Users\\u",
			want: filepath.Join("C:\\L", "cinedeck"),
		},
		{
			name: "windows home fallback",
			goos: "windows",
			env:  map[string]string{},
			home: "C:\\Users\\u",
			want: filepath.Join("C:\\Users\\u", "AppData", "Local", "cinedeck"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dataDirFor(tt.goos, fakeGetenv(tt.env), tt.home)
			if err != nil {
				t.Fatalf("dataDirFor: %v", err)
			}
			if got != tt.want {
				t.Errorf("dataDirFor = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDataDirForNoHome(t *testing.T) {
	_, err := dataDirFor("linux", fakeGetenv(map[string]string{}), "")
	if err == nil {
		t.Fatal("dataDirFor = nil, want error")
	}
	if !strings.Contains(err.Error(), "CINEDECK_DATA_DIR") {
		t.Errorf("error = %q, want it to contain %q", err, "CINEDECK_DATA_DIR")
	}
}

func TestDataDirAndLibraryPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CINEDECK_DATA_DIR", dir)

	got, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	if got != dir {
		t.Errorf("DataDir = %q, want %q", got, dir)
	}

	path, err := LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	if want := filepath.Join(dir, "library.db"); path != want {
		t.Errorf("LibraryPath = %q, want %q", path, want)
	}
}
