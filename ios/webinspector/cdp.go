package webinspector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type CDPServer struct {
	client *Client
	host   string
	port   int
	server *http.Server
}

type CDPTarget struct {
	Description          string `json:"description"`
	ID                   string `json:"id"`
	Title                string `json:"title"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	DevtoolsFrontendURL  string `json:"devtoolsFrontendUrl"`
}

func NewCDPServer(client *Client, host string, port int) *CDPServer {
	if host == "" {
		host = "127.0.0.1"
	}
	if port == 0 {
		port = 9222
	}
	return &CDPServer{client: client, host: host, port: port}
}

func (s *CDPServer) Addr() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

func (s *CDPServer) Serve(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/json", s.handleTargets)
	mux.HandleFunc("/json/list", s.handleTargets)
	mux.HandleFunc("/json/version", s.handleVersion)
	mux.HandleFunc("/devtools/page/", s.handlePageWebSocket)
	s.server = &http.Server{Addr: s.Addr(), Handler: mux}

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func (s *CDPServer) handleTargets(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	pages, err := s.client.ListPages(ctx, 250*time.Millisecond)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	targets := make([]CDPTarget, 0, len(pages))
	for _, appPage := range pages {
		page := appPage.Page
		if page.Type != WIRTypeWeb && page.Type != WIRTypeWebPage {
			continue
		}
		wsURL := fmt.Sprintf("ws://%s/devtools/page/%s", s.Addr(), page.Key)
		targets = append(targets, CDPTarget{
			ID:                   page.Key,
			Title:                page.Title,
			Type:                 "page",
			URL:                  page.URL,
			WebSocketDebuggerURL: wsURL,
			DevtoolsFrontendURL:  "/devtools/inspector.html?ws=" + s.Addr() + "/devtools/page/" + page.Key,
		})
	}
	writeJSON(w, targets)
}

func (s *CDPServer) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"Browser":              "Safari",
		"Protocol-Version":     "1.1",
		"User-Agent":           "go-ios",
		"WebKit-Version":       "",
		"webSocketDebuggerUrl": fmt.Sprintf("ws://%s/devtools/browser/%s", s.Addr(), s.client.connectionID),
	})
}

func (s *CDPServer) handlePageWebSocket(w http.ResponseWriter, r *http.Request) {
	pageKey := strings.TrimPrefix(r.URL.Path, "/devtools/page/")
	app, page, ok := s.client.FindPage(pageKey)
	if !ok {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	sessionID := strings.ToUpper(uuid.New().String())
	if err := s.client.SetupInspectorSocket(sessionID, app, page, false); err != nil {
		_ = ws.WriteJSON(cdpError(0, err))
		return
	}
	targetID := waitForTargetID(ctx, s.client, ws)
	if targetID == "" {
		return
	}

	errCh := make(chan error, 2)
	go func() {
		errCh <- s.forwardDeviceEvents(ctx, ws)
	}()
	go func() {
		errCh <- s.forwardBrowserCommands(ctx, ws, sessionID, app, page, targetID)
	}()
	<-errCh
}

func (s *CDPServer) forwardDeviceEvents(ctx context.Context, ws *websocket.Conn) error {
	for {
		event, err := s.client.NextEvent(ctx)
		if err != nil {
			return err
		}
		if dispatch, ok := unwrapDispatchMessage(event); ok {
			if err := ws.WriteJSON(dispatch); err != nil {
				return err
			}
			continue
		}
		if err := ws.WriteJSON(event); err != nil {
			return err
		}
	}
}

func (s *CDPServer) forwardBrowserCommands(ctx context.Context, ws *websocket.Conn, sessionID string, app Application, page Page, targetID string) error {
	for {
		var message map[string]any
		if err := ws.ReadJSON(&message); err != nil {
			return err
		}
		id, _ := numericInt(message["id"])
		method, _ := message["method"].(string)

		if shouldReplyLocally(method) {
			if err := ws.WriteJSON(map[string]any{"id": id, "result": map[string]any{}}); err != nil {
				return err
			}
			continue
		}

		wrapped := map[string]any{
			"method": "Target.sendMessageToTarget",
			"params": map[string]any{
				"targetId": targetID,
				"message":  mustJSON(message),
			},
		}
		if _, err := s.client.SendCommand(ctx, sessionID, app, page, nextWIRID(), "Target.sendMessageToTarget", wrapped["params"].(map[string]any)); err != nil {
			if writeErr := ws.WriteJSON(cdpError(id, err)); writeErr != nil {
				return writeErr
			}
		}
	}
}

func waitForTargetID(ctx context.Context, client *Client, ws *websocket.Conn) string {
	for {
		event, err := client.NextEvent(ctx)
		if err != nil {
			_ = ws.WriteJSON(cdpError(0, err))
			return ""
		}
		targetInfo, ok := event["params"].(map[string]any)["targetInfo"].(map[string]any)
		if !ok {
			continue
		}
		targetID, _ := targetInfo["targetId"].(string)
		_ = ws.WriteJSON(event)
		return targetID
	}
}

func unwrapDispatchMessage(event map[string]any) (map[string]any, bool) {
	if event["method"] != "Target.dispatchMessageFromTarget" {
		return nil, false
	}
	params, ok := event["params"].(map[string]any)
	if !ok {
		return nil, false
	}
	message, _ := params["message"].(string)
	if message == "" {
		return nil, false
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(message), &decoded); err != nil {
		return nil, false
	}
	return decoded, true
}

func shouldReplyLocally(method string) bool {
	switch method {
	case "Target.setDiscoverTargets",
		"Target.setAutoAttach",
		"Target.setRemoteLocations",
		"DOM.enable",
		"Log.enable",
		"Network.enable",
		"Page.enable",
		"Runtime.enable":
		return true
	default:
		return false
	}
}

func cdpError(id int, err error) map[string]any {
	return map[string]any{
		"id": id,
		"error": map[string]any{
			"code":    -32000,
			"message": err.Error(),
		},
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func mustJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func nextWIRID() int {
	return int(time.Now().UnixNano() & 0x7fffffff)
}
