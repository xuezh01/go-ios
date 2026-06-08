//go:build e2e

package preios17_test

import (
	"testing"
	"time"
)

// TestAccessibilityAudit launches a content-rich system app and runs the
// accessibility audit against it, asserting the device reports at least one
// issue and that each carries an issue type. Exercises the iOS 14 audit path
// (named case IDs via deviceBeginAuditCaseIDs:).
func TestAccessibilityAudit(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) {
		runIOSForDevice(t, udid, "launch", knownSystemApp)
		time.Sleep(2 * time.Second) // let the app settle into the foreground
		smokeArr(t, udid, []string{"issueType"}, "ax", "audit")
	})
}
