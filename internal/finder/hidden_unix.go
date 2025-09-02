//go:build !windows

package finder

import "strings"

// isHidden reports whether the entry should be considered hidden on Unix-like systems.
// We ignore the path parameter on Unix; dotfiles are the hidden convention here.
func isHidden(_ /*path*/ string, name string) bool {
	return strings.HasPrefix(name, ".")
}
