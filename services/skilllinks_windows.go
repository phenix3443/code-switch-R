//go:build windows

package services

import "fmt"

func createPlatformLink(target, linkPath string) error {
	return fmt.Errorf("windows junction not implemented")
}
