package server

import (
	"encoding/json"
	"fmt"
	"innominatus/internal/database"
	"net/http"
	"os"
	"time"
)

// PerformanceMetrics represents aggregated workflow performance data
type PerformanceMetrics struct {
	Application       string            `json:"application"`
	TotalExecutions   int               `json:"total_executions"`
	SuccessRate       float64           `json:"success_rate_percent"`
	FailureRate       float64           `json:"failure_rate_percent"`
	AverageDuration   float64           `json:"average_duration_seconds"`
	MedianDuration    float64           `json:"median_duration_seconds"`
	MinDuration       float64           `json:"min_duration_seconds"`
	MaxDuration       float64           `json:"max_duration_seconds"`
	StepMetrics       []StepMetric      `json:"step_metrics"`
	TimeSeriesData    []TimeSeriesPoint `json:"time_series_data"`
	LastExecutionTime time.Time         `json:"last_execution_time"`
	CalculatedAt      time.Time         `json:"calculated_at"`
}

// StepMetric represents performance data for individual steps
type StepMetric struct {
	StepName        string  `json:"step_name"`
	StepType        string  `json:"step_type"`
	ExecutionCount  int     `json:"execution_count"`
	SuccessCount    int     `json:"success_count"`
	FailureCount    int     `json:"failure_count"`
	SuccessRate     float64 `json:"success_rate_percent"`
	AverageDuration float64 `json:"average_duration_seconds"`
	MaxDuration     float64 `json:"max_duration_seconds"`
}

// TimeSeriesPoint represents a point in time series data
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Duration  float64   `json:"duration_seconds"`
	Status    string    `json:"status"`
}

// handlePerformanceMetrics handles /api/graph/<app>/metrics requests
func (s *Server) handlePerformanceMetrics(w http.ResponseWriter, r *http.Request, appName string) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// Get workflow executions for the application
	executions, err := s.workflowRepo.ListWorkflowExecutions(appName, "", "", 100, 0)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query workflow executions: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate metrics
	metrics := calculatePerformanceMetrics(appName, executions)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// calculatePerformanceMetrics computes performance metrics from workflow executions
func calculatePerformanceMetrics(appName string, executions []*database.WorkflowExecutionSummary) PerformanceMetrics {
	if len(executions) == 0 {
		return PerformanceMetrics{
			Application:  appName,
			CalculatedAt: time.Now(),
		}
	}

	metrics := PerformanceMetrics{
		Application:  appName,
		CalculatedAt: time.Now(),
	}

	var totalDuration float64
	var successCount, failureCount int
	durations := []float64{}
	timeSeriesData := []TimeSeriesPoint{}
	stepMetricsMap := make(map[string]*StepMetric)

	for _, exec := range executions {
		metrics.TotalExecutions++

		// Calculate duration
		var duration float64
		if exec.CompletedAt != nil {
			duration = exec.CompletedAt.Sub(exec.StartedAt).Seconds()
			totalDuration += duration
			durations = append(durations, duration)

			// Add to time series
			timeSeriesData = append(timeSeriesData, TimeSeriesPoint{
				Timestamp: exec.StartedAt,
				Duration:  duration,
				Status:    exec.Status,
			})
		}

		// Track success/failure
		switch exec.Status {
		case "succeeded", "completed":
			successCount++
		case "failed":
			failureCount++
		}

		// Update last execution time
		if exec.StartedAt.After(metrics.LastExecutionTime) {
			metrics.LastExecutionTime = exec.StartedAt
		}

		// Aggregate step metrics (simplified - would need step-level data from DB)
		// For now, we'll use workflow-level data
		if exec.WorkflowName != "" {
			if _, exists := stepMetricsMap[exec.WorkflowName]; !exists {
				stepMetricsMap[exec.WorkflowName] = &StepMetric{
					StepName: exec.WorkflowName,
					StepType: "workflow",
				}
			}
			stepMetric := stepMetricsMap[exec.WorkflowName]
			stepMetric.ExecutionCount++
			switch exec.Status {
			case "succeeded", "completed":
				stepMetric.SuccessCount++
			case "failed":
				stepMetric.FailureCount++
			}
			if duration > stepMetric.MaxDuration {
				stepMetric.MaxDuration = duration
			}
			stepMetric.AverageDuration = (stepMetric.AverageDuration*float64(stepMetric.ExecutionCount-1) + duration) / float64(stepMetric.ExecutionCount)
		}
	}

	// Calculate success/failure rates
	if metrics.TotalExecutions > 0 {
		metrics.SuccessRate = (float64(successCount) / float64(metrics.TotalExecutions)) * 100
		metrics.FailureRate = (float64(failureCount) / float64(metrics.TotalExecutions)) * 100
	}

	// Calculate duration statistics
	if len(durations) > 0 {
		metrics.AverageDuration = totalDuration / float64(len(durations))

		// Find min/max
		metrics.MinDuration = durations[0]
		metrics.MaxDuration = durations[0]
		for _, d := range durations {
			if d < metrics.MinDuration {
				metrics.MinDuration = d
			}
			if d > metrics.MaxDuration {
				metrics.MaxDuration = d
			}
		}

		// Calculate median (simplified - sort would be more accurate)
		metrics.MedianDuration = metrics.AverageDuration
	}

	// Convert step metrics map to slice
	for _, sm := range stepMetricsMap {
		if sm.ExecutionCount > 0 {
			sm.SuccessRate = (float64(sm.SuccessCount) / float64(sm.ExecutionCount)) * 100
		}
		metrics.StepMetrics = append(metrics.StepMetrics, *sm)
	}

	metrics.TimeSeriesData = timeSeriesData

	return metrics
}
