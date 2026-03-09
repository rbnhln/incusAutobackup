package util

import (
	"os"
	"path/filepath"
)

// CachePath returns the directory that incus auto backup should use for caching assets. If INCUSAUTOBACKUP_DIR is
// set, this path is $INCUSAUTOBACKUP_DIR/cache, otherwise it is /var/cache/incusAutobackup.
func CachePath(path ...string) string {
	varDir := os.Getenv("INCUSAUTOBACKUP_DIR")
	cacheDir := "/var/cache/incusAutobackup"
	if varDir != "" {
		cacheDir = filepath.Join(varDir, "cache")
	}

	items := make([]string, 0, len(path)+1)
	items = append(items, cacheDir)
	items = append(items, path...)
	return filepath.Join(items...)
}

// LogPath returns the directory that incus auto backup should put logs under. If INCUSAUTOBACKUP_DIR is
// set, this path is $INCUSAUTOBACKUP_DIR/logs, otherwise it is /var/log.
func LogPath(path ...string) string {
	varDir := os.Getenv("INCUSAUTOBACKUP_DIR")
	logDir := "/var/log"
	if varDir != "" {
		logDir = filepath.Join(varDir, "logs")
	}

	items := make([]string, 0, len(path)+1)
	items = append(items, logDir)
	items = append(items, path...)
	return filepath.Join(items...)
}

// RunPath returns the directory that incus auto backup should put runtime data under.
// If INCUSAUTOBACKUP_DIR is set, this path is $INCUSAUTOBACKUP_DIR/run, otherwise it is /run/incusAutobackup.
func RunPath(path ...string) string {
	varDir := os.Getenv("INCUSAUTOBACKUP_DIR")
	runDir := "/run/incusAutobackup"
	if varDir != "" {
		runDir = filepath.Join(varDir, "run")
	}

	items := make([]string, 0, len(path)+1)
	items = append(items, runDir)
	items = append(items, path...)
	return filepath.Join(items...)
}

// VarPath returns the provided path elements joined by a slash and
// appended to the end of $INCUSAUTOBACKUP_DIR, which defaults to /var/lib/incusAutobackup.
func VarPath(path ...string) string {
	varDir := os.Getenv("INCUSAUTOBACKUP_DIR")
	if varDir == "" {
		varDir = "/var/lib/incusAutobackup"
	}

	items := make([]string, 0, len(path)+1)
	items = append(items, varDir)
	items = append(items, path...)
	return filepath.Join(items...)
}

// SharePath returns the directory that incus auto backup should put static content under.
// If INCUSAUTOBACKUP_DIR is set, this path is $INCUSAUTOBACKUP_DIR/share, otherwise it is /usr/share/incusAutobackup.
func SharePath(path ...string) string {
	varDir := os.Getenv("INCUSAUTOBACKUP_DIR")
	usrDir := "/usr/share/incusAutobackup"
	if varDir != "" {
		usrDir = filepath.Join(varDir, "share")
	}

	items := make([]string, 0, len(path)+1)
	items = append(items, usrDir)
	items = append(items, path...)
	return filepath.Join(items...)
}

// UsrPath returns the directory that incus auto backup should put static library & binary content under.
// If INCUSAUTOBACKUP_DIR is set, this path is $INCUSAUTOBACKUP_DIR/lib, otherwise it is /usr/lib/incusAutobackup.
func UsrPath(path ...string) string {
	varDir := os.Getenv("INCUSAUTOBACKUP_DIR")
	usrDir := "/usr/lib/incusAutobackup"
	if varDir != "" {
		usrDir = filepath.Join(varDir, "lib")
	}

	items := make([]string, 0, len(path)+1)
	items = append(items, usrDir)
	items = append(items, path...)
	return filepath.Join(items...)
}

// IsDir returns true if the given path is a directory.
func IsDir(name string) bool {
	stat, err := os.Stat(name)
	if err != nil {
		return false
	}

	return stat.IsDir()
}

// IsUnixSocket returns true if the given path is either a Unix socket
// or a symbolic link pointing at a Unix socket.
func IsUnixSocket(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}

	return (stat.Mode() & os.ModeSocket) == os.ModeSocket
}
