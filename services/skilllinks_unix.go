//go:build !windows

package services

import "os"

func createPlatformLink(target, linkPath string) error {
	return os.Symlink(target, linkPath)
}
