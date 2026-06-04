//go:build e2e

package preios17_test

import (
	"encoding/json"
	"testing"
)

// boolField asserts "<cmd> get" returns field as a boolean.
func boolField(t *testing.T, udid, field, cmd string) {
	t.Helper()
	m := smokeObj(t, udid, []string{field}, cmd, "get")
	if _, ok := m[field].(bool); !ok {
		t.Fatalf("%s get: %s is not a bool: %v", cmd, field, m[field])
	}
}

func TestAssistivetouchGet(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) { boolField(t, udid, "AssistiveTouchEnabled", "assistivetouch") })
}

func TestVoiceoverGet(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) { boolField(t, udid, "VoiceOverTouchEnabled", "voiceover") })
}

func TestZoomGet(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) { boolField(t, udid, "ZoomTouchEnabled", "zoom") })
}

// TestDevmodeGet reads Developer Mode. On pre-iOS16 it does not exist as a
// toggle and is reported disabled, so assert only that the field is a bool —
// not that it is enabled (unlike the iOS 17+ suites).
func TestDevmodeGet(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) {
		m := smokeObj(t, udid, []string{"DeveloperModeEnabled"}, "devmode", "get")
		if _, ok := m["DeveloperModeEnabled"].(bool); !ok {
			t.Fatalf("devmode get: DeveloperModeEnabled is not a bool: %v", m["DeveloperModeEnabled"])
		}
	})
}

func TestTimeformatGet(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) {
		m := smokeObj(t, udid, []string{"TimeFormat"}, "timeformat", "get")
		switch m["TimeFormat"] {
		case "12h", "24h":
		default:
			t.Fatalf("timeformat get: TimeFormat = %v, want 12h or 24h", m["TimeFormat"])
		}
	})
}

// toggleRoundTrip reads a boolean accessibility setting, flips it, asserts the
// change, and always restores the original value.
func toggleRoundTrip(t *testing.T, udid, cmd, field string) {
	t.Helper()
	get := func() bool {
		var m map[string]any
		if err := json.Unmarshal(runIOSForDevice(t, udid, cmd, "get"), &m); err != nil {
			t.Fatalf("%s get: parse: %v", cmd, err)
		}
		b, ok := m[field].(bool)
		if !ok {
			t.Fatalf("%s get: missing bool field %q", cmd, field)
		}
		return b
	}
	set := func(on bool) {
		op := "disable"
		if on {
			op = "enable"
		}
		runIOSForDevice(t, udid, cmd, op)
	}

	orig := get()
	defer set(orig)
	set(!orig)
	if get() == orig {
		t.Fatalf("%s: state did not change after setting %v", cmd, !orig)
	}
}

func TestAssistivetouchToggle(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) { toggleRoundTrip(t, udid, "assistivetouch", "AssistiveTouchEnabled") })
}

func TestVoiceoverToggle(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) { toggleRoundTrip(t, udid, "voiceover", "VoiceOverTouchEnabled") })
}

func TestZoomToggle(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) { toggleRoundTrip(t, udid, "zoom", "ZoomTouchEnabled") })
}

// TestTimeformatRoundTrip sets the time format to the opposite value, asserts
// the change, and restores the original.
func TestTimeformatRoundTrip(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) {
		get := func() string {
			var m map[string]string
			if err := json.Unmarshal(runIOSForDevice(t, udid, "timeformat", "get"), &m); err != nil {
				t.Fatalf("timeformat get: parse: %v", err)
			}
			return m["TimeFormat"]
		}
		orig := get()
		other := "24h"
		if orig == "24h" {
			other = "12h"
		}
		defer runIOSForDevice(t, udid, "timeformat", orig)
		runIOSForDevice(t, udid, "timeformat", other)
		if got := get(); got != other {
			t.Fatalf("timeformat: set %s but got %s", other, got)
		}
	})
}
