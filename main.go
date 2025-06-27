package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"syscall"

	"github.com/mark3labs/mcp-go/server"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/api"
	"github.com/saswatamcode/promql-mcp/pkg/prompts"
	"github.com/saswatamcode/promql-mcp/pkg/tools"
)

const (
	serverInstructions = `Welcome to the PromQL MCP server!

You can use this server to interact with a Prometheus-compatible API or TSDB, but only for the purposes of generating queries.
This server does not support querying metrics or series directly, but rather focuses on helping you construct valid PromQL queries.

You can use the tool prometheus_get_series to query the series available in the Prometheus instance. This will help you understand the actual available metrics and their labels
and allow you to construct valid PromQL queries based on that information.

The user can ask a variety of questions related to health, kube pods, questions around specific workloads and so on. Try to use tools/prompts from this server
to generate accurate PromQL queries.`
	serverVersion = "0.1.0"
	serverName    = "promql-mcp"
)

var (
	apiURL       string
	mcpServerURL string
	logLevel     string
	stdio        bool
)

func init() {
	flag.StringVar(&apiURL, "api-url", "http://localhost:9090", "The Prometheus-compatible API URL")
	flag.StringVar(&mcpServerURL, "mcp-server-url", ":8080", "The MCP server URL")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.BoolVar(&stdio, "stdio", false, "Use stdio transport")
	flag.Parse()

	logHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: getLogLevel(logLevel),
	})
	slog.SetDefault(slog.New(logHandler))
}

func main() {
	slog.Info("Prometheus-compatible API URL configured", "url", apiURL)
	slog.Info("Log level set to", "level", logLevel)

	client, err := api.NewClient(api.Config{
		Address: apiURL,
	})
	if err != nil {
		slog.Error("Error creating Prometheus client", "error", err)
		os.Exit(1)
	}

	mcpServer := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
		server.WithInstructions(serverInstructions),
	)

	mcpServer.AddTool(tools.GetSeries(client))
	mcpServer.AddPrompt(prompts.GeneratePromQL(client))
	mcpServer.AddPrompt(prompts.GeneratePersesDashboard())

	ctx, cancel := context.WithCancel(context.Background())

	var g run.Group
	{
		g.Add(run.SignalHandler(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM))
	}
	{
		if stdio {
			slog.Info("Starting PromQL MCP server using stdio transport")
			stdioServer := server.NewStdioServer(mcpServer)
			g.Add(func() error {
				return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
			}, func(_ error) {
				cancel()
			})
		} else {
			slog.Info("Starting PromQL MCP server using Streamable HTTP transport on " + mcpServerURL)
			httpServer := server.NewStreamableHTTPServer(mcpServer)
			g.Add(func() error {
				return httpServer.Start(mcpServerURL)
			}, func(_ error) {
				_ = httpServer.Shutdown(ctx)
				cancel()
			})
		}
	}

	if err := g.Run(); err != nil {
		slog.Error("Error starting run group", "error", err)
		os.Exit(1)
	}
}

func getLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
