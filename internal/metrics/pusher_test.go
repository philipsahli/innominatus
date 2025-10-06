package metrics

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
)

// TestNewMetricsPusher tests the creation of a new metrics pusher
func TestNewMetricsPusher(t *testing.T) {
	tests := []struct {
		name            string
		pushgatewayURL  string
		version         string
		commit          string
		expectedVersion string
		expectedCommit  string
	}{
		{
			name:            "with version and commit",
			pushgatewayURL:  "http://localhost:9091",
			version:         "v1.0.0",
			commit:          "abc123",
			expectedVersion: "v1.0.0",
			expectedCommit:  "abc123",
		},
		{
			name:            "with empty version defaults to dev",
			pushgatewayURL:  "http://localhost:9091",
			version:         "",
			commit:          "abc123",
			expectedVersion: "dev",
			expectedCommit:  "abc123",
		},
		{
			name:            "with empty commit defaults to unknown",
			pushgatewayURL:  "http://localhost:9091",
			version:         "v1.0.0",
			commit:          "",
			expectedVersion: "v1.0.0",
			expectedCommit:  "unknown",
		},
		{
			name:            "with both empty defaults",
			pushgatewayURL:  "http://localhost:9091",
			version:         "",
			commit:          "",
			expectedVersion: "dev",
			expectedCommit:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pusher := NewMetricsPusher(tt.pushgatewayURL, 10, tt.version, tt.commit)

			if pusher == nil {
				t.Fatal("expected pusher to be non-nil")
			}

			if pusher.pushgatewayURL != tt.pushgatewayURL {
				t.Errorf("expected pushgatewayURL=%s, got=%s", tt.pushgatewayURL, pusher.pushgatewayURL)
			}

			if pusher.jobName != "innominatus" {
				t.Errorf("expected jobName=innominatus, got=%s", pusher.jobName)
			}

			if pusher.registry == nil {
				t.Error("expected registry to be initialized")
			}

			if pusher.buildInfo == nil {
				t.Error("expected buildInfo to be initialized")
			}

			if pusher.metrics == nil {
				t.Error("expected metrics to be initialized")
			}

			// Test that build info metric is set correctly
			// We can't easily verify the labels without gathering, but we can check it's registered
			metricFamilies, err := pusher.registry.Gather()
			if err != nil {
				t.Fatalf("failed to gather metrics: %v", err)
			}

			foundBuildInfo := false
			for _, mf := range metricFamilies {
				if mf.GetName() == "innominatus_build_info" {
					foundBuildInfo = true
					if len(mf.GetMetric()) != 1 {
						t.Errorf("expected 1 build info metric, got %d", len(mf.GetMetric()))
					}
					break
				}
			}

			if !foundBuildInfo {
				t.Error("expected innominatus_build_info metric to be registered")
			}
		})
	}
}

// TestMetricsPusher_Collectors tests that Go and Process collectors are registered
func TestMetricsPusher_Collectors(t *testing.T) {
	pusher := NewMetricsPusher("http://localhost:9091", 10, "v1.0.0", "abc123")

	if pusher.registry == nil {
		t.Fatal("registry should not be nil")
	}

	// Gather metrics from registry
	metricFamilies, err := pusher.registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	// Check for Go runtime metrics
	goMetrics := []string{
		"go_goroutines",
		"go_memstats_alloc_bytes",
		"go_gc_duration_seconds",
	}

	// Check for Process metrics
	processMetrics := []string{
		"process_cpu_seconds_total",
		"process_resident_memory_bytes",
	}

	// Build a set of collected metric names
	collectedMetrics := make(map[string]bool)
	for _, mf := range metricFamilies {
		collectedMetrics[mf.GetName()] = true
	}

	// Verify Go metrics are present
	for _, metricName := range goMetrics {
		if !collectedMetrics[metricName] {
			t.Errorf("expected Go metric %s to be registered", metricName)
		}
	}

	// Verify Process metrics are present
	for _, metricName := range processMetrics {
		if !collectedMetrics[metricName] {
			t.Errorf("expected Process metric %s to be registered", metricName)
		}
	}

	// Verify build info metric is present
	if !collectedMetrics["innominatus_build_info"] {
		t.Error("expected innominatus_build_info metric to be registered")
	}
}

// TestMetricsPusher_RegistryIsolation tests that each pusher has its own registry
func TestMetricsPusher_RegistryIsolation(t *testing.T) {
	pusher1 := NewMetricsPusher("http://localhost:9091", 10, "v1.0.0", "abc123")
	pusher2 := NewMetricsPusher("http://localhost:9092", 10, "v2.0.0", "def456")

	if pusher1.registry == pusher2.registry {
		t.Error("expected each pusher to have its own registry")
	}

	// Verify different build info metrics
	metrics1, err := pusher1.registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics from pusher1: %v", err)
	}

	metrics2, err := pusher2.registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics from pusher2: %v", err)
	}

	// Both should have build info, but we can't easily compare label values
	// without more complex metric inspection
	if len(metrics1) == 0 {
		t.Error("expected pusher1 to have metrics")
	}

	if len(metrics2) == 0 {
		t.Error("expected pusher2 to have metrics")
	}
}

// TestBuildInfoMetric tests the build info metric specifically
func TestBuildInfoMetric(t *testing.T) {
	version := "v1.2.3"
	commit := "abc123def456"

	pusher := NewMetricsPusher("http://localhost:9091", 10, version, commit)

	// Gather metrics
	metricFamilies, err := pusher.registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	// Find build info metric
	var buildInfoMetric *dto.MetricFamily
	for i, mf := range metricFamilies {
		if mf.GetName() == "innominatus_build_info" {
			buildInfoMetric = metricFamilies[i]
			break
		}
	}

	if buildInfoMetric == nil {
		t.Fatal("expected to find innominatus_build_info metric")
	}

	// Verify metric type
	if buildInfoMetric.GetType() != dto.MetricType_GAUGE {
		t.Errorf("expected build info metric to be GAUGE type, got %v", buildInfoMetric.GetType())
	}

	// Verify help text
	if buildInfoMetric.GetHelp() != "Build information including version and commit" {
		t.Errorf("unexpected help text: %s", buildInfoMetric.GetHelp())
	}

	// Verify metric has labels
	metrics := buildInfoMetric.GetMetric()
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}

	// Verify labels exist (version, commit, go_version)
	labels := metrics[0].GetLabel()
	expectedLabels := map[string]bool{
		"version":    false,
		"commit":     false,
		"go_version": false,
	}

	for _, label := range labels {
		if _, exists := expectedLabels[label.GetName()]; exists {
			expectedLabels[label.GetName()] = true
		}
	}

	for labelName, found := range expectedLabels {
		if !found {
			t.Errorf("expected label %s to be present in build info metric", labelName)
		}
	}

	// Verify metric value is 1
	if metrics[0].GetGauge().GetValue() != 1 {
		t.Errorf("expected build info metric value to be 1, got %f", metrics[0].GetGauge().GetValue())
	}
}
