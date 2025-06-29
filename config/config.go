// Package config handles embedded configuration files
package config

import (
	_ "embed"
)

//go:embed rclone.conf
var RcloneConfig []byte

// GetRcloneConfig returns the embedded rclone configuration
func GetRcloneConfig() []byte {
	return RcloneConfig
}
