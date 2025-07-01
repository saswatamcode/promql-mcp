package tools

import (
	"context"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const (
	GetSeriesToolDescription = `Allows you to get only series from Prometheus by querying the api/v1/series endpoint with a match param that is fully constructed PromQL expr.
An example output of this tool would be like the following,

We have the following series:

{__name__="some_metric", container="some_container"...}
...

You can actually use this tool to figure out what metrics are available within the Prometheus instance.
With this knowledge, you can then choose to optionally generate PromQL queries to give they user the data they want or to answer their question.
DO NOT try to get ALL series from this tool using match params like __name__=~\".*\".`
)

func GetSeries(client api.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("prometheus_get_series",
			mcp.WithDescription(GetSeriesToolDescription),
			mcp.WithString("match", mcp.Required(),
				mcp.Description("A fully constructed PromQL expr to match the series that will be sent as a match[] arg to the api/v1/series endpoint."))),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			match, ok := args["match"].(string)
			if !ok {
				return mcp.NewToolResultError("invalid type for 'match', expected string"), nil
			}

			v1api := v1.NewAPI(client)
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			lblSets, warnings, err := v1api.Series(ctx, []string{match}, time.Now().Add(-time.Hour), time.Now())
			if err != nil {
				slog.Error("error querying Prometheus", "error", err)
				return mcp.NewToolResultError("error querying Prometheus: " + err.Error()), err
			}
			if len(warnings) > 0 {
				slog.Warn("Prometheus warnings", "warnings", warnings)
			}

			txt := "We have the following series:\n\n"

			for _, lblSet := range lblSets {
				txt += lblSet.String() + "\n"
			}

			return mcp.NewToolResultText(txt), nil
		}
}
