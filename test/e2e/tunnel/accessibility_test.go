//go:build e2e

package tunnel_test

import (
	"testing"
	"time"
)

// TestAccessibilityAudit launches a content-rich system app and runs the
// accessibility audit against it, asserting the device reports at least one
// issue and that each carries an issue type. The audit service itself is reached
// over usbmuxd, but bringing an app to the foreground on iOS 17+ needs the
// tunnel, so the test lives in this suite. Exercises the iOS 15+ audit path
// (integer categories via deviceBeginAuditTypes:).
func TestAccessibilityAudit(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) {
		runIOSForDevice(t, udid, "launch", "com.apple.Preferences")
		time.Sleep(2 * time.Second) // let the app settle into the foreground
		smokeArr(t, udid, []string{"issueType"}, "ax", "audit")
	})
}
