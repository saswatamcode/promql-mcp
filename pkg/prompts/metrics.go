package prompts

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/prometheus/client_golang/api"
)

const (
	GeneratePromQLPrompt = `
Think of yourself as a PromQL Expert SRE who is well versed in the Prometheus/Kubernetes ecosystem and open source.
I want you to generate a PromQL query to answer the user's question the best way possible.

Use prometheus_get_series tool to get the list of metrics that are available to query within the TSDB.
You can use this tool multiple times if needed, and actually use the output from this tool. DO NOT generate a query without using this tool.
Make sure that whatever query you generate, is valid according the output from this tool.

Ensure that,
- The PromQL query is valid PromQL and will not cause errors and can actually run,.
- The PromQL query is URL encodable.
- The PromQL query takes into account the upstream and open source best practices and norms for Prometheus.
- The PromQL query make reasonable assumptions from the query and the metrics provided as well as their nomenclature.
- Ensure that your final PromQL query has balanced brackets and balanced double quotes(when dealing with label selectors)

Now for the output, first, explain what the query does and how it helps answer the question. 
Then, on a new line, provide just the PromQL query between <PROMQL> and </PROMQL> tags.
Also provide a query URL for that query right after that. Assume that the promethes is available at %s.
For mulitple queries, provide a new line after each query.

Format your response like this:
Your explanation of what the query does and how it helps...

<PROMQL>your_query_here</PROMQL>
http://%s/api/v1/query?query=your_query_here
...

And finally here is the user's actual question: %s`
)

func GeneratePromQL(client api.Client) (prompt mcp.Prompt, handler server.PromptHandlerFunc) {
	return mcp.NewPrompt("prometheus_generate_promql",
			mcp.WithPromptDescription("A detailed prompt to generate a PromQL query to answer the user's question the best way possible."),
			mcp.WithArgument("question", mcp.RequiredArgument(), mcp.ArgumentDescription("The original user's question.")),
		),
		func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			question, ok := request.Params.Arguments["question"]
			if !ok {
				return nil, errors.New("question is required")
			}

			apiURL := client.URL("", map[string]string{})
			prompt := fmt.Sprintf(GeneratePromQLPrompt, apiURL.String(), apiURL.String(), question)

			return mcp.NewGetPromptResult(
				"A detailed prompt to generate a PromQL query to answer the user's question the best way possible.",
				[]mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.NewTextContent(prompt),
					},
				},
			), nil
		}
}
