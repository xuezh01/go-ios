//go:build e2e

package preios17_test

import (
	"testing"
	"time"
)

// TestSyslog streams the device syslog (lockdown syslog_relay, no tunnel) and
// asserts output is produced within the window.
func TestSyslog(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) {
		streamSmoke(t, udid, 8*time.Second, "syslog")
	})
}

// TestOstrace streams os_trace_relay logs and asserts output within the window.
func TestOstrace(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) {
		streamSmoke(t, udid, 8*time.Second, "ostrace")
	})
}
