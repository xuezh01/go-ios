package webinspector

import "testing"

func TestParseApplication(t *testing.T) {
	app, err := parseApplication(map[string]any{
		"WIRApplicationIdentifierKey":       "PID:123",
		"WIRApplicationBundleIdentifierKey": "com.apple.mobilesafari",
		"WIRApplicationNameKey":             "Safari",
		"WIRAutomationAvailabilityKey":      "WIRAutomationAvailabilityAvailable",
		"WIRIsApplicationActiveKey":         true,
		"WIRIsApplicationProxyKey":          false,
		"WIRIsApplicationReadyKey":          true,
	})
	if err != nil {
		t.Fatalf("parse application: %v", err)
	}
	if app.PID != 123 {
		t.Fatalf("expected pid 123, got %d", app.PID)
	}
	if app.BundleID != "com.apple.mobilesafari" {
		t.Fatalf("unexpected bundle id: %s", app.BundleID)
	}
}

func TestParseWebPage(t *testing.T) {
	page, err := parsePage("1", map[string]any{
		"WIRPageIdentifierKey": uint64(1),
		"WIRTypeKey":           "WIRTypeWebPage",
		"WIRTitleKey":          "Example",
		"WIRURLKey":            "https://example.test/",
	})
	if err != nil {
		t.Fatalf("parse page: %v", err)
	}
	if page.ID != 1 || page.Type != WIRTypeWebPage || page.Title != "Example" {
		t.Fatalf("unexpected page: %#v", page)
	}
}

func TestDecodeDispatchMessage(t *testing.T) {
	decoded, ok := decodeDispatchMessage(map[string]any{
		"method": "Target.dispatchMessageFromTarget",
		"params": map[string]any{
			"message": `{"id":7,"result":{"ok":true}}`,
		},
	})
	if !ok {
		t.Fatal("expected dispatch message to decode")
	}
	if id, _ := numericInt(decoded["id"]); id != 7 {
		t.Fatalf("expected id 7, got %d", id)
	}
}
