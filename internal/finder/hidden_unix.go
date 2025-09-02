// internal/finder/hidden_unix.go
//go:build !windows

package finder

import "strings"

func isHidden(path, name string) bool {
	return strings.HasPrefix(name, ".")
}
