// Package hl provides an adapter for the hl (https://github.com/pamburus/hl) log viewer.
// It enables logviewer to use hl as a high-performance backend for filtering and
// displaying logs when available.
package hl

import (
	"os/exec"
	"sync"
)

var (
	hlAvailable     bool
	hlChecked       bool
	hlCheckMu       sync.Mutex
	hlPath          string
	hlDetectDisable bool // For testing: allows disabling hl detection
)

// IsAvailable checks if the hl binary is available in the system PATH.
// The result is cached after the first check.
func IsAvailable() bool {
	hlCheckMu.Lock()
	defer hlCheckMu.Unlock()

	if hlDetectDisable {
		return false
	}

	if hlChecked {
		return hlAvailable
	}

	path, err := exec.LookPath("hl")
	if err == nil && path != "" {
		hlAvailable = true
		hlPath = path
	} else {
		hlAvailable = false
	}
	hlChecked = true

	return hlAvailable
}

// GetPath returns the full path to the hl binary if available.
// Returns empty string if hl is not found.
func GetPath() string {
	if !IsAvailable() {
		return ""
	}
	return hlPath
}

// Reset clears the cached availability check. Useful for testing.
func Reset() {
	hlCheckMu.Lock()
	defer hlCheckMu.Unlock()
	hlChecked = false
	hlAvailable = false
	hlPath = ""
}

// DisableDetection disables hl detection (forces native Go engine).
// This is useful for testing or when users want to force the native engine.
func DisableDetection() {
	hlCheckMu.Lock()
	defer hlCheckMu.Unlock()
	hlDetectDisable = true
}

// EnableDetection re-enables hl detection.
func EnableDetection() {
	hlCheckMu.Lock()
	defer hlCheckMu.Unlock()
	hlDetectDisable = false
	hlChecked = false // Force re-check on next IsAvailable()
}
