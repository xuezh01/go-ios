//go:build e2e

package preios17_test

import "testing"

// The commands below are part of the go-ios feature set but cannot be asserted
// unattended on these pre-iOS17 devices, each for a concrete reason. They are
// kept here (skipped, with the reason) so the gap is explicit rather than
// silently absent.

// sysmontap streams CPU/memory samples over instruments. On these old devices
// it connects but emits no samples within a bounded window (verified: nothing
// in 15s), so there is nothing deterministic to assert.
func TestSysmontap(t *testing.T) {
	t.Skip("sysmontap emits no samples within a bounded window on pre-iOS17 devices")
}

// instruments notifications only emits on app lifecycle transitions, so it
// produces nothing without separately driving an app — not unattended-assertable
// here (launch/kill is covered by TestLaunchKill).
func TestInstrumentsNotifications(t *testing.T) {
	t.Skip("instruments notifications only emits on app lifecycle events; nothing to assert unattended")
}

// pcap/ip depend on the device's pcapd and on live traffic: pcap fails outright
// on iOS 12.5.7 and ip blocks until it observes the device's IP in the capture.
// pcap is covered against a modern device in the tunnel-free suite (test/e2e).
func TestPcap(t *testing.T) {
	t.Skip("pcap is version/traffic-dependent on pre-iOS17 (fails on iOS 12.5.7); covered in test/e2e")
}

func TestIP(t *testing.T) {
	t.Skip("ip blocks until it detects the device IP from live capture; not deterministic unattended")
}
