//go:build !windows

package prowlarr

import "os/exec"

func hideWindow(cmd *exec.Cmd) {} // no-op on non-Windows
