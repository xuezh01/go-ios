//go:build e2e

package preios17_test

import "testing"

// knownSystemApp is always installed, so the apps tests can assert it is listed.
const knownSystemApp = "com.apple.Preferences"

// TestApps lists system apps in compact text form and asserts a known system
// app appears.
func TestApps(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) {
		smokeContains(t, udid, knownSystemApp, "apps", "--system", "--list")
	})
}

// TestAppsAll lists all apps as JSON and asserts a known system app is present.
func TestAppsAll(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) {
		apps := smokeArr(t, udid, []string{"CFBundleIdentifier", "CFBundleName", "ApplicationType"}, "apps", "--all")
		for _, a := range apps {
			if m, ok := a.(map[string]any); ok && m["CFBundleIdentifier"] == knownSystemApp {
				return
			}
		}
		t.Fatalf("apps --all: %s not found in app list", knownSystemApp)
	})
}

// TestFsyncTree lists the AFC media directory tree (lockdown/usbmux, no tunnel).
func TestFsyncTree(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) { smoke(t, udid, "fsync", "tree", "--path=/") })
}
