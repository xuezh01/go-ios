//go:build e2e

package e2e_test

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestList(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) {
		out := runIOS(t, "list")

		var v struct {
			DeviceList []string `json:"deviceList"`
		}
		if err := json.Unmarshal(out, &v); err != nil {
			t.Fatalf("parse: %v\n%s", err, out)
		}
		if !slices.Contains(v.DeviceList, udid) {
			t.Fatalf("device %s not present in list: %v", udid, v.DeviceList)
		}
	})
}

// TestListDetails checks `list --details` reports the device with a real
// ConnectionType (never the "unknown" fallback) and that its identity fields
// match the snapshot. A device can be listed more than once (e.g. once over USB
// and once over Network), so every entry for the udid is validated and the
// recorded ConnectionType must appear among them.
func TestListDetails(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) {
		out := runIOS(t, "list", "--details")

		var v struct {
			DeviceList []map[string]any `json:"deviceList"`
		}
		if err := json.Unmarshal(out, &v); err != nil {
			t.Fatalf("parse: %v\n%s", err, out)
		}

		var entries []map[string]any
		for _, d := range v.DeviceList {
			if u, _ := d["Udid"].(string); u == udid {
				entries = append(entries, d)
			}
		}
		if len(entries) == 0 {
			t.Fatalf("device %s not present in list --details: %s", udid, out)
		}

		exp, known := expectedDevice(udid)
		seen := map[string]bool{}
		for _, d := range entries {
			ct, _ := d["ConnectionType"].(string)
			if ct != "USB" && ct != "Network" {
				t.Fatalf("device %s ConnectionType = %q, want USB or Network", udid, ct)
			}
			seen[ct] = true
			// Identity is stable across connection types; assert it on every entry.
			if known {
				for _, key := range []string{"ProductName", "ProductType", "ProductVersion"} {
					if got, _ := d[key].(string); got != exp[key] {
						t.Fatalf("%s = %q, want %q (test/e2e/testdata/devices.json)", key, got, exp[key])
					}
				}
			}
		}
		if known {
			if want := exp["ConnectionType"]; want != "" && !seen[want] {
				t.Fatalf("device %s has no %q connection in list --details (saw %v)", udid, want, seen)
			}
		}
	})
}
