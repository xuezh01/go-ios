package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/danielpaulus/go-ios/ios/webinspector"
)

func runWebInspectorCommand(cmdCtx commandContext) {
	timeoutSeconds, _ := cmdCtx.Args.String("--timeout")
	timeout := parseWebInspectorTimeout(timeoutSeconds)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, err := webinspector.New(cmdCtx.Device)
	exitIfError("failed connecting to webinspector", err)
	defer client.Close()
	exitIfError("failed starting webinspector", client.Connect(ctx))

	if list, _ := cmdCtx.Args.Bool("list"); list {
		pages, err := client.ListPages(ctx, 500*time.Millisecond)
		exitIfError("failed listing webinspector pages", err)
		fmt.Println(convertToJSONString(pages))
		return
	}

	if eval, _ := cmdCtx.Args.Bool("eval"); eval {
		pageID, _ := cmdCtx.Args.String("<pageID>")
		expression, _ := cmdCtx.Args.String("<expression>")
		app, page := selectWebInspectorPage(ctx, client, pageID)
		result, err := client.Evaluate(ctx, app, page, expression)
		exitIfError("failed evaluating JavaScript", err)
		fmt.Println(convertToJSONString(result))
		return
	}

	if cdp, _ := cmdCtx.Args.Bool("cdp"); cdp {
		host, _ := cmdCtx.Args.String("--host")
		port, _ := cmdCtx.Args.Int("--port")
		if port == 0 {
			port = 9222
		}
		server := webinspector.NewCDPServer(client, host, port)
		slog.Info("webinspector CDP server started", "addr", server.Addr())
		serverCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		err := server.Serve(serverCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			exitIfError("webinspector CDP server failed", err)
		}
		return
	}
}

func selectWebInspectorPage(ctx context.Context, client *webinspector.Client, pageID string) (webinspector.Application, webinspector.Page) {
	if pageID != "" {
		app, page, ok := client.FindPage(pageID)
		if ok {
			return app, page
		}
	}
	pages, err := client.ListPages(ctx, 500*time.Millisecond)
	exitIfError("failed listing webinspector pages", err)
	for _, candidate := range pages {
		if candidate.Page.Type != webinspector.WIRTypeWeb && candidate.Page.Type != webinspector.WIRTypeWebPage && candidate.Page.Type != webinspector.WIRTypeJavaScript {
			continue
		}
		if pageID == "" || candidate.Page.Key == pageID {
			return candidate.Application, candidate.Page
		}
	}
	exitIfError("failed finding webinspector page", fmt.Errorf("page %q not found", pageID))
	return webinspector.Application{}, webinspector.Page{}
}

func parseWebInspectorTimeout(raw string) time.Duration {
	if raw == "" {
		return 5 * time.Second
	}
	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil || seconds <= 0 {
		return 5 * time.Second
	}
	return time.Duration(seconds * float64(time.Second))
}
