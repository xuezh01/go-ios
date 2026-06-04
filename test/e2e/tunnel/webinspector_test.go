//go:build e2e

package tunnel_test

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/webinspector"
	"github.com/gorilla/websocket"
)

func TestWebInspectorBrowserControl(t *testing.T) {
	forEachDevice(t, func(t *testing.T, udid string) {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		client, app, page := openWebInspectorFixture(t, ctx, udid)

		value, err := client.Evaluate(ctx, app, page, "document.title")
		if err != nil {
			t.Fatalf("webinspector evaluate document.title: %v", err)
		}
		if value != "go-ios-e2e" {
			t.Fatalf("document.title = %#v, want go-ios-e2e", value)
		}

		port := freeTCPPort(t)
		server := webinspector.NewCDPServer(client, "127.0.0.1", port)
		serverCtx, stopServer := context.WithCancel(ctx)
		defer stopServer()
		errCh := make(chan error, 1)
		go func() { errCh <- server.Serve(serverCtx) }()
		waitForCDPServer(t, server.Addr())

		ws := connectCDPPage(t, server.Addr(), page.Key)
		defer ws.Close()

		cdpCommand(t, ws, 1, "Runtime.evaluate", map[string]any{
			"expression":    `document.body.innerHTML = '<input id="agent" value="">'; document.getElementById("agent").focus();`,
			"returnByValue": true,
		})
		cdpCommand(t, ws, 2, "Input.dispatchKeyEvent", map[string]any{
			"type": "char",
			"key":  "a",
			"text": "a",
		})
		valueResponse := cdpCommand(t, ws, 3, "Runtime.evaluate", map[string]any{
			"expression":    `document.getElementById("agent").value`,
			"returnByValue": true,
		})
		if got := cdpEvaluateValue(valueResponse); got != "a" {
			t.Fatalf("input value after CDP key event = %#v, want a", got)
		}

		nodeResponse := cdpCommand(t, ws, 4, "DOM.getNodeForLocation", map[string]any{"x": 1, "y": 1})
		nodeResult, _ := nodeResponse["result"].(map[string]any)
		if nodeID, ok := numeric(nodeResult["nodeId"]); !ok || nodeID == 0 {
			t.Fatalf("DOM.getNodeForLocation returned no nodeId: %#v", nodeResponse)
		}

		objectResponse := cdpCommand(t, ws, 5, "Runtime.evaluate", map[string]any{
			"expression": `document.getElementById("agent")`,
		})
		objectID := cdpEvaluateObjectID(objectResponse)
		if objectID == "" {
			t.Fatalf("Runtime.evaluate returned no objectId: %#v", objectResponse)
		}
		listenersResponse := cdpCommand(t, ws, 6, "DOMDebugger.getEventListeners", map[string]any{"objectId": objectID})
		listenersResult, _ := listenersResponse["result"].(map[string]any)
		if _, ok := listenersResult["listeners"].([]any); !ok {
			t.Fatalf("DOMDebugger.getEventListeners returned no listeners array: %#v", listenersResponse)
		}

		resourceTree := cdpCommand(t, ws, 7, "Page.getResourceTree", map[string]any{})
		resourceResult, _ := resourceTree["result"].(map[string]any)
		if _, ok := resourceResult["frameTree"].(map[string]any); !ok {
			t.Fatalf("Page.getResourceTree returned no frameTree: %#v", resourceTree)
		}

		cdpCommand(t, ws, 8, "Page.startScreencast", map[string]any{
			"format":    "jpeg",
			"quality":   80,
			"maxWidth":  400,
			"maxHeight": 400,
		})
		frame := cdpEvent(t, ws, "Page.screencastFrame")
		frameParams, _ := frame["params"].(map[string]any)
		if data, _ := frameParams["data"].(string); data == "" {
			t.Fatalf("Page.screencastFrame has empty data: %#v", frame)
		}
		sessionID, _ := numeric(frameParams["sessionId"])
		cdpCommand(t, ws, 9, "Page.screencastFrameAck", map[string]any{"sessionId": sessionID})
		cdpCommand(t, ws, 10, "Page.stopScreencast", map[string]any{})

		_ = ws.Close()
		stopServer()
		select {
		case err := <-errCh:
			if err != nil && !strings.Contains(err.Error(), "context canceled") {
				t.Fatalf("cdp server: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("cdp server did not stop")
		}
	})
}

func openWebInspectorFixture(t *testing.T, ctx context.Context, udid string) (*webinspector.Client, webinspector.Application, webinspector.Page) {
	t.Helper()

	device, err := ios.GetDevice(udid)
	if err != nil {
		t.Fatalf("get device: %v", err)
	}
	client, err := webinspector.New(device)
	if err != nil {
		t.Fatalf("connect webinspector service: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Connect(ctx); err != nil {
		skipWebInspectorSetting(t, err)
		t.Fatalf("start webinspector: %v", err)
	}

	app, err := client.OpenApp(ctx, webinspector.SafariBundleID)
	if err != nil {
		t.Fatalf("open Safari: %v", err)
	}
	session, err := client.AutomationSession(ctx, app)
	if err != nil {
		skipWebInspectorSetting(t, err)
		t.Fatalf("start remote automation: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = session.Stop(stopCtx)
	})
	if err := session.Start(ctx); err != nil {
		t.Fatalf("start browsing context: %v", err)
	}
	url := "data:text/html,%3Ctitle%3Ego-ios-e2e%3C/title%3E%3Cinput%20id%3Dagent%20value%3D%22%22%3E"
	if err := session.Navigate(ctx, url); err != nil {
		t.Fatalf("navigate fixture page: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		pages, err := client.ListPages(ctx, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("list webinspector pages: %v", err)
		}
		for _, candidate := range pages {
			if candidate.Application.BundleID != webinspector.SafariBundleID {
				continue
			}
			if candidate.Page.Type != webinspector.WIRTypeWeb && candidate.Page.Type != webinspector.WIRTypeWebPage {
				continue
			}
			if strings.Contains(candidate.Page.URL, "go-ios-e2e") || candidate.Page.Title == "go-ios-e2e" {
				return client, candidate.Application, candidate.Page
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("fixture page did not appear in WebInspector listing")
	return nil, webinspector.Application{}, webinspector.Page{}
}

func skipWebInspectorSetting(t *testing.T, err error) {
	t.Helper()
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "web inspector is not enabled") ||
		strings.Contains(message, "remote automation is not enabled") {
		t.Skip(err)
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on free port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func waitForCDPServer(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/json/version")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("cdp server %s did not become ready", addr)
}

func connectCDPPage(t *testing.T, addr string, pageKey string) *websocket.Conn {
	t.Helper()
	ws, _, err := websocket.DefaultDialer.Dial("ws://"+addr+"/devtools/page/"+pageKey, nil)
	if err != nil {
		t.Fatalf("connect CDP websocket: %v", err)
	}
	return ws
}

func cdpCommand(t *testing.T, ws *websocket.Conn, id int, method string, params map[string]any) map[string]any {
	t.Helper()
	if err := ws.WriteJSON(map[string]any{"id": id, "method": method, "params": params}); err != nil {
		t.Fatalf("write CDP command %s: %v", method, err)
	}
	for {
		message := readCDPMessage(t, ws)
		if gotID, ok := numeric(message["id"]); ok && gotID == id {
			if errValue, ok := message["error"]; ok {
				t.Fatalf("CDP command %s failed: %#v", method, errValue)
			}
			return message
		}
	}
}

func cdpEvent(t *testing.T, ws *websocket.Conn, method string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		message := readCDPMessage(t, ws)
		if message["method"] == method {
			return message
		}
	}
	t.Fatalf("timed out waiting for CDP event %s", method)
	return nil
}

func readCDPMessage(t *testing.T, ws *websocket.Conn) map[string]any {
	t.Helper()
	_ = ws.SetReadDeadline(time.Now().Add(10 * time.Second))
	var message map[string]any
	if err := ws.ReadJSON(&message); err != nil {
		t.Fatalf("read CDP message: %v", err)
	}
	return message
}

func cdpEvaluateValue(response map[string]any) any {
	result, _ := response["result"].(map[string]any)
	remote, _ := result["result"].(map[string]any)
	return remote["value"]
}

func cdpEvaluateObjectID(response map[string]any) string {
	result, _ := response["result"].(map[string]any)
	remote, _ := result["result"].(map[string]any)
	objectID, _ := remote["objectId"].(string)
	return objectID
}

func numeric(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}
