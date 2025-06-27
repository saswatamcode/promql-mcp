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

	GeneratePersesDashboardPrompt = `
Think of yourself as a PromQL Expert SRE who is well versed in the Prometheus/Kubernetes ecosystem and open source.
I want you to generate PromQL queries to answer the user's question in the most holistic way possible.

Use prometheus_get_series tool to get the list of metrics that are available to query within the TSDB.
You can use this tool multiple times if needed, and actually use the output from this tool. DO NOT generate a query without using this tool.
Make sure that whatever query you generate, is valid according the output from this tool.

Ensure that,
- The PromQL query is valid PromQL and will not cause errors and can actually run,.
- The PromQL query is URL encodable.
- The PromQL query takes into account the upstream and open source best practices and norms for Prometheus.
- The PromQL query make reasonable assumptions from the query and the metrics provided as well as their nomenclature.
- Ensure that your final PromQL query has balanced brackets and balanced double quotes(when dealing with label selectors)

Generate a PersesDashboard Kubernetes CR object that contains the queries in the panels.
Accurately determine the type of the panel based on the query and the question. You can reconsider queries and fit them into what you think is the best panel type
to represent that information.
Consider that a human SRE will actually be looking at this dashboard, so make sure that the panels are actually relevant and helpful, for quick, and accurate decision making
during incidents.

Ensure to accurately fill out the datasource as %s and the namespace as %s, and use best practice kubernetes labels for it as well.
Ensure that you use the proper variables for the dashboards and proper PromQL for the same as well. Ensure that you retrofit that variable into the PromQL you generate.

Format your response within a YAML markdown codeblock.
Here is a qualified example of a PersesDashboard object that you can use as a reference:
apiVersion: perses.dev/v1alpha1
kind: PersesDashboard
metadata:
  creationTimestamp: null
  labels:
    app.kubernetes.io/component: dashboard
    app.kubernetes.io/instance: kubernetes-cluster-resources-overview
    app.kubernetes.io/name: perses-dashboard
    app.kubernetes.io/part-of: perses-operator
  name: kubernetes-cluster-resources-overview
  namespace: perses-dev
spec:
  display:
    name: Kubernetes / Compute Resources / Cluster
  duration: 1h
  layouts:
  - kind: Grid
    spec:
      display:
        title: Cluster Stats
      items:
      - content:
          $ref: '#/spec/panels/0_0'
        height: 4
        width: 24
        x: 0
        "y": 0
  - kind: Grid
    spec:
      display:
        title: CPU Usage
      items:
      - content:
          $ref: '#/spec/panels/1_0'
        height: 8
        width: 24
        x: 0
        "y": 0
  - kind: Grid
    spec:
      display:
        title: Storage IO - Distribution
      items:
      - content:
          $ref: '#/spec/panels/2_0'
        height: 10
        width: 24
        x: 0
        "y": 0
  panels:
    "0_0":
      kind: Panel
      spec:
        display:
          description: Shows the CPU utilization of the cluster.
          name: CPU Utilization
        plugin:
          kind: StatChart
          spec:
            calculation: last
            format:
              decimalPlaces: 2
              unit: percent
            valueFontSize: 50
        queries:
        - kind: TimeSeriesQuery
          spec:
            plugin:
              kind: PrometheusTimeSeriesQuery
              spec:
                datasource:
                  kind: PrometheusDatasource
                  name: custom-datasource
                query: cluster:node_cpu:ratio_rate5m{cluster="$cluster"}
    "1_0":
      kind: Panel
      spec:
        display:
          description: Shows the CPU usage of the cluster by namespace.
          name: CPU Usage
        plugin:
          kind: TimeSeriesChart
          spec:
            legend:
              mode: list
              position: bottom
              size: small
            visual:
              areaOpacity: 1
              display: line
              lineWidth: 0.25
              palette:
                mode: auto
            yAxis:
              format:
                unit: decimal
        queries:
        - kind: TimeSeriesQuery
          spec:
            plugin:
              kind: PrometheusTimeSeriesQuery
              spec:
                datasource:
                  kind: PrometheusDatasource
                  name: custom-datasource
                query: |-
                  sum by (namespace) (
                    node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate5m{cluster="$cluster"}
                  )
                seriesNameFormat: '{{namespace}}'
    "2_0":
      kind: Panel
      spec:
        display:
          description: Shows the current storage IO of the cluster in tabular form, by namespace.
          name: Current Storage IO
        plugin:
          kind: Table
          spec:
            columnSettings:
            - align: left
              format:
                unit: ""
              header: Namespace
              name: namespace
            - align: right
              format:
                unit: ops/sec
              header: IOPS(Reads)
              name: 'value #1'
            - align: right
              format:
                unit: ops/sec
              header: IOPS(Writes)
              name: 'value #2'
            - align: right
              format:
                unit: ops/sec
              header: IOPS(Reads + Writes)
              name: 'value #3'
            - align: right
              format:
                unit: bytes/sec
              header: Throughput(Reads)
              name: 'value #4'
            - align: right
              format:
                unit: bytes/sec
              header: Throughput(Writes)
              name: 'value #5'
            - align: right
              format:
                unit: bytes/sec
              header: Throughput(Reads + Writes)
              name: 'value #6'
            - format:
                unit: ""
              hide: true
              name: timestamp
            transforms:
            - kind: MergeSeries
              spec: {}
        queries:
        - kind: TimeSeriesQuery
          spec:
            plugin:
              kind: PrometheusTimeSeriesQuery
              spec:
                datasource:
                  kind: PrometheusDatasource
                  name: custom-datasource
                query: |-
                  sum by (namespace) (
                    rate(
                      container_fs_reads_total{cluster="$cluster",container!="",device=~"(/dev.+)|mmcblk.p.+|nvme.+|rbd.+|sd.+|vd.+|xvd.+|dm-.+|dasd.+",job="cadvisor",namespace!=""}[$__rate_interval]
                    )
                  )
        - kind: TimeSeriesQuery
          spec:
            plugin:
              kind: PrometheusTimeSeriesQuery
              spec:
                datasource:
                  kind: PrometheusDatasource
                  name: custom-datasource
                query: |-
                  sum by (namespace) (
                    rate(
                      container_fs_writes_total{cluster="$cluster",container!="",device=~"(/dev.+)|mmcblk.p.+|nvme.+|rbd.+|sd.+|vd.+|xvd.+|dm-.+|dasd.+",job="cadvisor",namespace!=""}[$__rate_interval]
                    )
                  )
        - kind: TimeSeriesQuery
          spec:
            plugin:
              kind: PrometheusTimeSeriesQuery
              spec:
                datasource:
                  kind: PrometheusDatasource
                  name: custom-datasource
                query: |-
                  sum by (namespace) (
                      rate(
                        container_fs_reads_total{cluster="$cluster",container!="",device=~"(/dev.+)|mmcblk.p.+|nvme.+|rbd.+|sd.+|vd.+|xvd.+|dm-.+|dasd.+",job="cadvisor",namespace!=""}[$__rate_interval]
                      )
                    +
                      rate(
                        container_fs_writes_total{cluster="$cluster",container!="",device=~"(/dev.+)|mmcblk.p.+|nvme.+|rbd.+|sd.+|vd.+|xvd.+|dm-.+|dasd.+",job="cadvisor",namespace!=""}[$__rate_interval]
                      )
                  )
        - kind: TimeSeriesQuery
          spec:
            plugin:
              kind: PrometheusTimeSeriesQuery
              spec:
                datasource:
                  kind: PrometheusDatasource
                  name: custom-datasource
                query: |-
                  sum by (namespace) (
                    rate(
                      container_fs_reads_bytes_total{cluster="$cluster",container!="",device=~"(/dev.+)|mmcblk.p.+|nvme.+|rbd.+|sd.+|vd.+|xvd.+|dm-.+|dasd.+",job="cadvisor",namespace!=""}[$__rate_interval]
                    )
                  )
        - kind: TimeSeriesQuery
          spec:
            plugin:
              kind: PrometheusTimeSeriesQuery
              spec:
                datasource:
                  kind: PrometheusDatasource
                  name: custom-datasource
                query: |-
                  sum by (namespace) (
                    rate(
                      container_fs_writes_bytes_total{cluster="$cluster",container!="",device=~"(/dev.+)|mmcblk.p.+|nvme.+|rbd.+|sd.+|vd.+|xvd.+|dm-.+|dasd.+",job="cadvisor",namespace!=""}[$__rate_interval]
                    )
                  )
        - kind: TimeSeriesQuery
          spec:
            plugin:
              kind: PrometheusTimeSeriesQuery
              spec:
                datasource:
                  kind: PrometheusDatasource
                  name: custom-datasource
                query: |-
                  sum by (namespace) (
                      rate(
                        container_fs_reads_bytes_total{cluster="$cluster",container!="",device=~"(/dev.+)|mmcblk.p.+|nvme.+|rbd.+|sd.+|vd.+|xvd.+|dm-.+|dasd.+",job="cadvisor",namespace!=""}[$__rate_interval]
                      )
                    +
                      rate(
                        container_fs_writes_bytes_total{cluster="$cluster",container!="",device=~"(/dev.+)|mmcblk.p.+|nvme.+|rbd.+|sd.+|vd.+|xvd.+|dm-.+|dasd.+",job="cadvisor",namespace!=""}[$__rate_interval]
                      )
                  )
  variables:
  - kind: ListVariable
    spec:
      allowAllValue: false
      allowMultiple: false
      display:
        hidden: false
        name: cluster
      name: cluster
      plugin:
        kind: PrometheusLabelValuesVariable
        spec:
          datasource:
            kind: PrometheusDatasource
            name: custom-datasource
          labelName: cluster
          matchers:
          - up{job="kubelet", metrics_path="/metrics/cadvisor"}
status: {}


And finally, here's the user's actual question: %s
`
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

func GeneratePersesDashboard() (prompt mcp.Prompt, handler server.PromptHandlerFunc) {
	return mcp.NewPrompt("perses_generate_dashboard",
			mcp.WithPromptDescription("A detailed prompt to generate a PersesDashboard object with fully qualified PromQL queries to answer the user's question the best way possible."),
			mcp.WithArgument("question", mcp.RequiredArgument(), mcp.ArgumentDescription("The original user's question.")),
			mcp.WithArgument("datasource", mcp.RequiredArgument(), mcp.ArgumentDescription("The datasource to use for the dashboard, e.g., prometheus.")),
			mcp.WithArgument("namespace_or_project", mcp.RequiredArgument(), mcp.ArgumentDescription("The namespace to use for the dashboard, e.g., default.")),
		),
		func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			question, ok := request.Params.Arguments["question"]
			if !ok {
				return nil, errors.New("question is required")
			}
			datasource, ok := request.Params.Arguments["datasource"]
			if !ok {
				return nil, errors.New("datasource is required")
			}
			namespace, ok := request.Params.Arguments["namespace_or_project"]
			if !ok {
				return nil, errors.New("namespace_or_project is required")
			}

			prompt := fmt.Sprintf(GeneratePersesDashboardPrompt, datasource, namespace, question)

			return mcp.NewGetPromptResult(
				"A detailed prompt to generate a PersesDashboard object to answer the user's question the best way possible.",
				[]mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.NewTextContent(prompt),
					},
				},
			), nil
		}
}
