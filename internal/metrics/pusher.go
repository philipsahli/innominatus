package metrics

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/push"
)

// MetricsPusher pushes metrics to Prometheus Pushgateway
type MetricsPusher struct {
	pushgatewayURL string
	pushInterval   time.Duration
	jobName        string
	stopChan       chan struct{}
	metrics        *Metrics
	registry       *prometheus.Registry
	buildInfo      *prometheus.GaugeVec
}

// NewMetricsPusher creates a new metrics pusher with runtime collectors
func NewMetricsPusher(pushgatewayURL string, pushInterval time.Duration, version, commit string) *MetricsPusher {
	// Create a new registry for all collectors
	registry := prometheus.NewRegistry()

	// Register standard Go runtime metrics collector
	registry.MustRegister(collectors.NewGoCollector())

	// Register process metrics collector (CPU, memory, file descriptors)
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Create build info metric
	buildInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "innominatus_build_info",
			Help: "Build information including version and commit",
		},
		[]string{"version", "commit", "go_version"},
	)

	// Set build info with current values
	if version == "" {
		version = "dev"
	}
	if commit == "" {
		commit = "unknown"
	}
	buildInfo.WithLabelValues(version, commit, runtime.Version()).Set(1)

	// Register build info
	registry.MustRegister(buildInfo)

	return &MetricsPusher{
		pushgatewayURL: pushgatewayURL,
		pushInterval:   pushInterval,
		jobName:        "innominatus",
		stopChan:       make(chan struct{}),
		metrics:        GetGlobal(),
		registry:       registry,
		buildInfo:      buildInfo,
	}
}

// StartPushing starts pushing metrics to the Pushgateway in a background goroutine
func (p *MetricsPusher) StartPushing() {
	go p.pushLoop()
	log.Printf("📊 Started pushing metrics to %s every %v", p.pushgatewayURL, p.pushInterval)
}

// Stop stops pushing metrics
func (p *MetricsPusher) Stop() {
	close(p.stopChan)
	log.Println("📊 Stopped pushing metrics")
}

// pushLoop continuously pushes metrics at the configured interval
func (p *MetricsPusher) pushLoop() {
	ticker := time.NewTicker(p.pushInterval)
	defer ticker.Stop()

	// Push immediately on start
	if err := p.pushMetrics(); err != nil {
		log.Printf("Failed to push metrics: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := p.pushMetrics(); err != nil {
				log.Printf("Failed to push metrics: %v", err)
			}
		case <-p.stopChan:
			return
		}
	}
}

// pushMetrics pushes all metrics to the Pushgateway
func (p *MetricsPusher) pushMetrics() error {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	pusher := push.New(p.pushgatewayURL, p.jobName)

	// Add the registry containing Go runtime and process collectors
	pusher.Gatherer(p.registry)

	// Add uptime gauge
	uptimeGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "innominatus_uptime_seconds",
		Help: "Server uptime in seconds",
	})
	uptimeGauge.Set(time.Since(p.metrics.startTime).Seconds())
	pusher.Collector(uptimeGauge)

	// Add workflow metrics
	// Note: We use gauges for counter metrics when pushing to Pushgateway
	workflowsExecutedGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "innominatus_workflows_executed_total",
		Help: "Total workflow executions",
	})
	workflowsExecutedGauge.Set(float64(p.metrics.workflowsExecuted))
	pusher.Collector(workflowsExecutedGauge)

	workflowsSucceededGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "innominatus_workflows_succeeded_total",
		Help: "Total successful workflow executions",
	})
	workflowsSucceededGauge.Set(float64(p.metrics.workflowsSucceeded))
	pusher.Collector(workflowsSucceededGauge)

	workflowsFailedGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "innominatus_workflows_failed_total",
		Help: "Total failed workflow executions",
	})
	workflowsFailedGauge.Set(float64(p.metrics.workflowsFailed))
	pusher.Collector(workflowsFailedGauge)

	// Average workflow duration
	if len(p.metrics.workflowDurations) > 0 {
		var total time.Duration
		for _, d := range p.metrics.workflowDurations {
			total += d
		}
		avgSeconds := (total / time.Duration(len(p.metrics.workflowDurations))).Seconds()
		avgDurationGauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "innominatus_workflow_duration_seconds_avg",
			Help: "Average workflow duration (last 100 executions)",
		})
		avgDurationGauge.Set(avgSeconds)
		pusher.Collector(avgDurationGauge)
	}

	// Database metrics
	dbQueriesGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "innominatus_db_queries_total",
		Help: "Total database queries",
	})
	dbQueriesGauge.Set(float64(p.metrics.dbQueriesTotal))
	pusher.Collector(dbQueriesGauge)

	dbErrorsGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "innominatus_db_query_errors_total",
		Help: "Total database query errors",
	})
	dbErrorsGauge.Set(float64(p.metrics.dbQueryErrors))
	pusher.Collector(dbErrorsGauge)

	// HTTP request metrics (aggregated - individual paths would create too many metrics)
	var totalHTTPRequests int64
	for _, paths := range p.metrics.httpRequestsTotal {
		for _, count := range paths {
			totalHTTPRequests += count
		}
	}
	httpRequestsGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "innominatus_http_requests_total",
		Help: "Total HTTP requests (all paths)",
	})
	httpRequestsGauge.Set(float64(totalHTTPRequests))
	pusher.Collector(httpRequestsGauge)

	var totalHTTPErrors int64
	for _, count := range p.metrics.httpRequestErrors {
		totalHTTPErrors += count
	}
	httpErrorsGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "innominatus_http_errors_total",
		Help: "Total HTTP 5xx errors",
	})
	httpErrorsGauge.Set(float64(totalHTTPErrors))
	pusher.Collector(httpErrorsGauge)

	// Push all metrics
	if err := pusher.Push(); err != nil {
		return fmt.Errorf("failed to push metrics to pushgateway: %w", err)
	}

	return nil
}
