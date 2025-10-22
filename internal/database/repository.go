package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"innominatus/internal/metrics"
	"time"
)

// WorkflowRepository handles database operations for workflows
type WorkflowRepository struct {
	db *Database
}

// NewWorkflowRepository creates a new workflow repository
func NewWorkflowRepository(db *Database) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

// CreateWorkflowExecution creates a new workflow execution record
func (r *WorkflowRepository) CreateWorkflowExecution(appName, workflowName string, totalSteps int) (*WorkflowExecution, error) {
	query := `
		INSERT INTO workflow_executions (application_name, workflow_name, status, total_steps, started_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, application_name, workflow_name, status, started_at, total_steps, created_at, updated_at
	`

	execution := &WorkflowExecution{}
	err := r.db.db.QueryRow(query, appName, workflowName, WorkflowStatusRunning, totalSteps).Scan(
		&execution.ID,
		&execution.ApplicationName,
		&execution.WorkflowName,
		&execution.Status,
		&execution.StartedAt,
		&execution.TotalSteps,
		&execution.CreatedAt,
		&execution.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create workflow execution: %w", err)
	}

	return execution, nil
}

// UpdateWorkflowExecution updates the workflow execution status
func (r *WorkflowRepository) UpdateWorkflowExecution(id int64, status string, errorMessage *string) error {
	var query string
	var args []interface{}

	if status == WorkflowStatusCompleted || status == WorkflowStatusFailed {
		query = `
			UPDATE workflow_executions
			SET status = $1, completed_at = NOW(), error_message = $2
			WHERE id = $3
		`
		args = []interface{}{status, errorMessage, id}
	} else {
		query = `
			UPDATE workflow_executions
			SET status = $1, error_message = $2
			WHERE id = $3
		`
		args = []interface{}{status, errorMessage, id}
	}

	_, err := r.db.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update workflow execution: %w", err)
	}

	return nil
}

// CreateWorkflowStep creates a new workflow step record
func (r *WorkflowRepository) CreateWorkflowStep(workflowID int64, stepNumber int, stepName, stepType string, stepConfig map[string]interface{}) (*WorkflowStepExecution, error) {
	configJSON, err := json.Marshal(stepConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal step config: %w", err)
	}

	query := `
		INSERT INTO workflow_step_executions (workflow_execution_id, step_number, step_name, step_type, status, step_config)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, workflow_execution_id, step_number, step_name, step_type, status, created_at, updated_at
	`

	step := &WorkflowStepExecution{}
	err = r.db.db.QueryRow(query, workflowID, stepNumber, stepName, stepType, StepStatusPending, configJSON).Scan(
		&step.ID,
		&step.WorkflowExecutionID,
		&step.StepNumber,
		&step.StepName,
		&step.StepType,
		&step.Status,
		&step.CreatedAt,
		&step.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create workflow step: %w", err)
	}

	step.StepConfig = stepConfig
	return step, nil
}

// UpdateWorkflowStepStatus updates the workflow step status
func (r *WorkflowRepository) UpdateWorkflowStepStatus(stepID int64, status string, errorMessage *string) error {
	var query string
	var args []interface{}

	now := time.Now()

	switch status {
	case StepStatusRunning:
		query = `
			UPDATE workflow_step_executions
			SET status = $1, started_at = $2, error_message = $3
			WHERE id = $4
		`
		args = []interface{}{status, now, errorMessage, stepID}
	case StepStatusCompleted, StepStatusFailed:
		// Calculate duration if step has started_at
		query = `
			UPDATE workflow_step_executions
			SET status = $1, completed_at = $2, error_message = $3,
			    duration_ms = CASE WHEN started_at IS NOT NULL
			                      THEN EXTRACT(EPOCH FROM ($2 - started_at)) * 1000
			                      ELSE NULL END
			WHERE id = $4
		`
		args = []interface{}{status, now, errorMessage, stepID}
	default:
		query = `
			UPDATE workflow_step_executions
			SET status = $1, error_message = $2
			WHERE id = $3
		`
		args = []interface{}{status, errorMessage, stepID}
	}

	_, err := r.db.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update workflow step status: %w", err)
	}

	// Record step metrics when step completes or fails
	if status == StepStatusCompleted || status == StepStatusFailed {
		// Fetch step info to get step type and duration
		var stepType string
		var durationMs sql.NullInt64
		err := r.db.db.QueryRow(`
			SELECT step_type, duration_ms
			FROM workflow_step_executions
			WHERE id = $1
		`, stepID).Scan(&stepType, &durationMs)

		if err == nil && durationMs.Valid {
			// Record step execution metrics
			metrics.GetGlobal().RecordWorkflowStep(stepType, status == StepStatusCompleted, durationMs.Int64)
		}
	}

	return nil
}

// AddWorkflowStepLogs adds output logs to a workflow step
func (r *WorkflowRepository) AddWorkflowStepLogs(stepID int64, logs string) error {
	query := `
		UPDATE workflow_step_executions
		SET output_logs = COALESCE(output_logs, '') || $1
		WHERE id = $2
	`

	_, err := r.db.db.Exec(query, logs, stepID)
	if err != nil {
		return fmt.Errorf("failed to add workflow step logs: %w", err)
	}

	return nil
}

// GetWorkflowExecution retrieves a workflow execution by ID
func (r *WorkflowRepository) GetWorkflowExecution(id int64) (*WorkflowExecution, error) {
	query := `
		SELECT id, application_name, workflow_name, status, started_at, completed_at,
		       error_message, total_steps, created_at, updated_at
		FROM workflow_executions
		WHERE id = $1
	`

	execution := &WorkflowExecution{}
	err := r.db.db.QueryRow(query, id).Scan(
		&execution.ID,
		&execution.ApplicationName,
		&execution.WorkflowName,
		&execution.Status,
		&execution.StartedAt,
		&execution.CompletedAt,
		&execution.ErrorMessage,
		&execution.TotalSteps,
		&execution.CreatedAt,
		&execution.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow execution not found")
		}
		return nil, fmt.Errorf("failed to get workflow execution: %w", err)
	}

	// Load steps
	steps, err := r.GetWorkflowSteps(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow steps: %w", err)
	}
	execution.Steps = steps

	return execution, nil
}

// GetWorkflowSteps retrieves all steps for a workflow execution
func (r *WorkflowRepository) GetWorkflowSteps(workflowID int64) ([]*WorkflowStepExecution, error) {
	query := `
		SELECT id, workflow_execution_id, step_number, step_name, step_type, status,
		       started_at, completed_at, duration_ms, error_message, step_config, output_logs,
		       created_at, updated_at
		FROM workflow_step_executions
		WHERE workflow_execution_id = $1
		ORDER BY step_number ASC
	`

	rows, err := r.db.db.Query(query, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow steps: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var steps []*WorkflowStepExecution
	for rows.Next() {
		step := &WorkflowStepExecution{}
		var stepConfigJSON []byte

		err := rows.Scan(
			&step.ID,
			&step.WorkflowExecutionID,
			&step.StepNumber,
			&step.StepName,
			&step.StepType,
			&step.Status,
			&step.StartedAt,
			&step.CompletedAt,
			&step.DurationMs,
			&step.ErrorMessage,
			&stepConfigJSON,
			&step.OutputLogs,
			&step.CreatedAt,
			&step.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow step: %w", err)
		}

		// Parse step config JSON
		if stepConfigJSON != nil {
			var config map[string]interface{}
			if err := json.Unmarshal(stepConfigJSON, &config); err == nil {
				step.StepConfig = config
			}
		}

		steps = append(steps, step)
	}

	return steps, nil
}

// CountWorkflowExecutions counts total workflow executions matching filters
func (r *WorkflowRepository) CountWorkflowExecutions(appName, workflowName, status string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM workflow_executions
		WHERE ($1 = '' OR application_name = $1)
		  AND ($2 = '' OR workflow_name ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR status = $3)
	`

	var count int64
	err := r.db.db.QueryRow(query, appName, workflowName, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count workflow executions: %w", err)
	}

	return count, nil
}

// ListWorkflowExecutions lists workflow executions with optional filtering
func (r *WorkflowRepository) ListWorkflowExecutions(appName, workflowName, status string, limit, offset int) ([]*WorkflowExecutionSummary, error) {
	query := `
		SELECT we.id, we.application_name, we.workflow_name, we.status, we.started_at,
		       we.completed_at, we.total_steps,
		       COALESCE(step_stats.completed_steps, 0) as completed_steps,
		       COALESCE(step_stats.failed_steps, 0) as failed_steps,
		       CASE WHEN we.completed_at IS NOT NULL
		            THEN CAST(EXTRACT(EPOCH FROM (we.completed_at - we.started_at)) * 1000 AS BIGINT)
		            ELSE NULL END as duration
		FROM workflow_executions we
		LEFT JOIN (
			SELECT workflow_execution_id,
			       COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_steps,
			       COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_steps
			FROM workflow_step_executions
			GROUP BY workflow_execution_id
		) step_stats ON we.id = step_stats.workflow_execution_id
		WHERE ($1 = '' OR we.application_name = $1)
		  AND ($2 = '' OR we.workflow_name ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR we.status = $3)
		ORDER BY we.started_at DESC
		LIMIT $4 OFFSET $5
	`
	args := []interface{}{appName, workflowName, status, limit, offset}

	rows, err := r.db.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow executions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var executions []*WorkflowExecutionSummary
	for rows.Next() {
		exec := &WorkflowExecutionSummary{}

		err := rows.Scan(
			&exec.ID,
			&exec.ApplicationName,
			&exec.WorkflowName,
			&exec.Status,
			&exec.StartedAt,
			&exec.CompletedAt,
			&exec.TotalSteps,
			&exec.CompletedSteps,
			&exec.FailedSteps,
			&exec.Duration,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow execution: %w", err)
		}

		executions = append(executions, exec)
	}

	return executions, nil
}

// GetLatestWorkflowExecution retrieves the most recent workflow execution for an app/workflow combination
func (r *WorkflowRepository) GetLatestWorkflowExecution(appName, workflowName string) (*WorkflowExecution, error) {
	query := `
		SELECT id, application_name, workflow_name, status, started_at, completed_at,
		       error_message, total_steps, created_at, updated_at,
		       parent_execution_id, retry_count, is_retry, resume_from_step
		FROM workflow_executions
		WHERE application_name = $1 AND workflow_name = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	execution := &WorkflowExecution{}
	err := r.db.db.QueryRow(query, appName, workflowName).Scan(
		&execution.ID,
		&execution.ApplicationName,
		&execution.WorkflowName,
		&execution.Status,
		&execution.StartedAt,
		&execution.CompletedAt,
		&execution.ErrorMessage,
		&execution.TotalSteps,
		&execution.CreatedAt,
		&execution.UpdatedAt,
		&execution.ParentExecutionID,
		&execution.RetryCount,
		&execution.IsRetry,
		&execution.ResumeFromStep,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No execution found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest workflow execution: %w", err)
	}

	return execution, nil
}

// GetFirstFailedStepNumber finds the step number of the first failed step in a workflow execution
func (r *WorkflowRepository) GetFirstFailedStepNumber(executionID int64) (int, error) {
	query := `
		SELECT step_number
		FROM workflow_step_executions
		WHERE workflow_execution_id = $1 AND status = $2
		ORDER BY step_number ASC
		LIMIT 1
	`

	var stepNumber int
	err := r.db.db.QueryRow(query, executionID, StepStatusFailed).Scan(&stepNumber)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("no failed step found")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get first failed step: %w", err)
	}

	return stepNumber, nil
}

// CreateRetryExecution creates a new workflow execution as a retry of a previous execution
func (r *WorkflowRepository) CreateRetryExecution(parentID int64, appName, workflowName string, totalSteps, resumeFromStep int) (*WorkflowExecution, error) {
	// Get parent execution to calculate retry count
	parent, err := r.GetWorkflowExecution(parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent execution: %w", err)
	}

	retryCount := parent.RetryCount + 1

	query := `
		INSERT INTO workflow_executions (
			application_name, workflow_name, status, total_steps, started_at,
			parent_execution_id, retry_count, is_retry, resume_from_step
		)
		VALUES ($1, $2, $3, $4, NOW(), $5, $6, $7, $8)
		RETURNING id, application_name, workflow_name, status, started_at, total_steps,
		          created_at, updated_at, parent_execution_id, retry_count, is_retry, resume_from_step
	`

	execution := &WorkflowExecution{}
	err = r.db.db.QueryRow(
		query,
		appName,
		workflowName,
		WorkflowStatusRunning,
		totalSteps,
		parentID,
		retryCount,
		true, // is_retry
		resumeFromStep,
	).Scan(
		&execution.ID,
		&execution.ApplicationName,
		&execution.WorkflowName,
		&execution.Status,
		&execution.StartedAt,
		&execution.TotalSteps,
		&execution.CreatedAt,
		&execution.UpdatedAt,
		&execution.ParentExecutionID,
		&execution.RetryCount,
		&execution.IsRetry,
		&execution.ResumeFromStep,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create retry execution: %w", err)
	}

	return execution, nil
}

// ReconstructWorkflowFromExecution reconstructs a workflow specification from stored step executions
// This allows retrying a workflow without requiring the original workflow file
func (r *WorkflowRepository) ReconstructWorkflowFromExecution(executionID int64) (map[string]interface{}, error) {
	// Get all steps for this execution, ordered by step number
	query := `
		SELECT step_number, step_name, step_type, step_config
		FROM workflow_step_executions
		WHERE workflow_execution_id = $1
		ORDER BY step_number ASC
	`

	rows, err := r.db.db.Query(query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow steps: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var steps []map[string]interface{}

	for rows.Next() {
		var stepNumber int
		var stepName, stepType string
		var stepConfigJSON sql.NullString

		if err := rows.Scan(&stepNumber, &stepName, &stepType, &stepConfigJSON); err != nil {
			return nil, fmt.Errorf("failed to scan step row: %w", err)
		}

		// Parse step_config JSONB
		var stepConfig map[string]interface{}
		if stepConfigJSON.Valid && stepConfigJSON.String != "" {
			if err := json.Unmarshal([]byte(stepConfigJSON.String), &stepConfig); err != nil {
				return nil, fmt.Errorf("failed to unmarshal step config for step %d: %w", stepNumber, err)
			}
		} else {
			stepConfig = make(map[string]interface{})
		}

		// Reconstruct step with name, type, and config fields
		step := map[string]interface{}{
			"name": stepName,
			"type": stepType,
		}

		// Merge step_config into step (this includes path, playbook, namespace, etc.)
		for k, v := range stepConfig {
			step[k] = v
		}

		steps = append(steps, step)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating step rows: %w", err)
	}

	if len(steps) == 0 {
		return nil, fmt.Errorf("no steps found for workflow execution %d", executionID)
	}

	// Reconstruct workflow structure
	workflow := map[string]interface{}{
		"steps": steps,
	}

	return workflow, nil
}
