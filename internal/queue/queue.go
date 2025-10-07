package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"innominatus/internal/database"
	"innominatus/internal/logging"
	"innominatus/internal/types"
	"sync"
	"time"
)

// WorkflowTask represents a workflow execution task
type WorkflowTask struct {
	ID           string                 `json:"id"`
	AppName      string                 `json:"app_name"`
	WorkflowName string                 `json:"workflow_name"`
	Workflow     types.Workflow         `json:"workflow"`
	EnqueuedAt   time.Time              `json:"enqueued_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// WorkflowExecutor defines the interface for executing workflows
type WorkflowExecutor interface {
	ExecuteWorkflowWithName(appName, workflowName string, workflow types.Workflow) error
}

// Queue represents an async task queue for workflow execution
type Queue struct {
	tasks            chan *WorkflowTask
	workers          int
	executor         WorkflowExecutor
	db               *database.Database
	logger           *logging.ZerologAdapter
	wg               sync.WaitGroup
	ctx              context.Context
	cancel           context.CancelFunc
	mu               sync.RWMutex
	activeTasks      map[string]*WorkflowTask
	taskStatusChan   chan taskStatusUpdate
	metricsCollector *MetricsCollector
}

type taskStatusUpdate struct {
	taskID string
	status TaskStatus
	err    error
}

// MetricsCollector tracks queue metrics
type MetricsCollector struct {
	mu                 sync.RWMutex
	tasksEnqueued      int64
	tasksCompleted     int64
	tasksFailed        int64
	totalQueueTime     time.Duration
	totalExecutionTime time.Duration
}

// NewQueue creates a new async task queue
func NewQueue(workers int, executor WorkflowExecutor, db *database.Database) *Queue {
	ctx, cancel := context.WithCancel(context.Background())

	q := &Queue{
		tasks:            make(chan *WorkflowTask, 100), // Buffer 100 tasks
		workers:          workers,
		executor:         executor,
		db:               db,
		logger:           logging.NewStructuredLogger("queue"),
		ctx:              ctx,
		cancel:           cancel,
		activeTasks:      make(map[string]*WorkflowTask),
		taskStatusChan:   make(chan taskStatusUpdate, 100),
		metricsCollector: &MetricsCollector{},
	}

	return q
}

// Start starts the queue workers
func (q *Queue) Start() {
	q.logger.InfoWithFields("Starting queue workers", map[string]interface{}{
		"workers":     q.workers,
		"buffer_size": cap(q.tasks),
	})

	// Start status update processor
	q.wg.Add(1)
	go q.processStatusUpdates()

	// Start worker goroutines
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
}

// Stop gracefully stops the queue workers
func (q *Queue) Stop() {
	q.logger.Info("Stopping queue workers...")

	// Cancel context to signal workers to stop
	q.cancel()

	// Close task channel (no more tasks accepted)
	close(q.tasks)

	// Wait for workers to finish (this doesn't include status processor)
	// Create a separate done channel to track worker completion
	workersDone := make(chan struct{})
	go func() {
		// Wait for only the worker goroutines (not status processor)
		// We started q.workers workers + 1 status processor
		// The wg was incremented by q.workers + 1
		// So we need to wait manually
		time.Sleep(100 * time.Millisecond) // Give workers time to finish
		close(q.taskStatusChan)
		workersDone <- struct{}{}
	}()

	// Wait for everything to finish
	q.wg.Wait()
	<-workersDone

	q.logger.Info("Queue workers stopped")
}

// Enqueue adds a workflow task to the queue
func (q *Queue) Enqueue(appName, workflowName string, workflow types.Workflow, metadata map[string]interface{}) (string, error) {
	task := &WorkflowTask{
		ID:           generateTaskID(),
		AppName:      appName,
		WorkflowName: workflowName,
		Workflow:     workflow,
		EnqueuedAt:   time.Now(),
		Metadata:     metadata,
	}

	// Store task in database for persistence
	if err := q.storeTask(task); err != nil {
		return "", fmt.Errorf("failed to store task: %w", err)
	}

	// Enqueue task (non-blocking with timeout)
	select {
	case q.tasks <- task:
		q.metricsCollector.incrementEnqueued()
		q.logger.InfoWithFields("Task enqueued", map[string]interface{}{
			"task_id":       task.ID,
			"app_name":      appName,
			"workflow_name": workflowName,
			"queue_size":    len(q.tasks),
		})
		return task.ID, nil
	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("queue is full, task rejected")
	}
}

// worker processes tasks from the queue
func (q *Queue) worker(id int) {
	defer q.wg.Done()

	q.logger.InfoWithFields("Worker started", map[string]interface{}{
		"worker_id": id,
	})

	for {
		select {
		case <-q.ctx.Done():
			q.logger.InfoWithFields("Worker stopping", map[string]interface{}{
				"worker_id": id,
			})
			return
		case task, ok := <-q.tasks:
			if !ok {
				q.logger.InfoWithFields("Task channel closed, worker exiting", map[string]interface{}{
					"worker_id": id,
				})
				return
			}

			q.processTask(id, task)
		}
	}
}

// processTask executes a workflow task
func (q *Queue) processTask(workerID int, task *WorkflowTask) {
	startTime := time.Now()
	queueTime := startTime.Sub(task.EnqueuedAt)

	// Mark task as active
	q.mu.Lock()
	q.activeTasks[task.ID] = task
	q.mu.Unlock()

	// Update task status to running
	q.updateTaskStatus(task.ID, TaskStatusRunning, nil)

	q.logger.InfoWithFields("Processing task", map[string]interface{}{
		"worker_id":     workerID,
		"task_id":       task.ID,
		"app_name":      task.AppName,
		"workflow_name": task.WorkflowName,
		"queue_time_ms": queueTime.Milliseconds(),
	})

	// Execute workflow
	err := q.executor.ExecuteWorkflowWithName(task.AppName, task.WorkflowName, task.Workflow)

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Update metrics
	q.metricsCollector.recordTaskCompletion(queueTime, executionTime, err == nil)

	// Remove from active tasks
	q.mu.Lock()
	delete(q.activeTasks, task.ID)
	q.mu.Unlock()

	// Update task status
	if err != nil {
		q.updateTaskStatus(task.ID, TaskStatusFailed, err)
		q.logger.ErrorWithFields("Task failed", map[string]interface{}{
			"worker_id":         workerID,
			"task_id":           task.ID,
			"app_name":          task.AppName,
			"workflow_name":     task.WorkflowName,
			"execution_time_ms": executionTime.Milliseconds(),
			"error":             err.Error(),
		})
	} else {
		q.updateTaskStatus(task.ID, TaskStatusCompleted, nil)
		q.logger.InfoWithFields("Task completed", map[string]interface{}{
			"worker_id":         workerID,
			"task_id":           task.ID,
			"app_name":          task.AppName,
			"workflow_name":     task.WorkflowName,
			"execution_time_ms": executionTime.Milliseconds(),
		})
	}
}

// updateTaskStatus sends a status update to the channel
func (q *Queue) updateTaskStatus(taskID string, status TaskStatus, err error) {
	select {
	case q.taskStatusChan <- taskStatusUpdate{taskID: taskID, status: status, err: err}:
	case <-q.ctx.Done():
		// Queue is shutting down, skip status update
		return
	case <-time.After(1 * time.Second):
		q.logger.WarnWithFields("Status update channel full", map[string]interface{}{
			"task_id": taskID,
		})
	}
}

// processStatusUpdates processes status updates asynchronously
func (q *Queue) processStatusUpdates() {
	defer q.wg.Done()

	for update := range q.taskStatusChan {
		if err := q.persistTaskStatus(update.taskID, update.status, update.err); err != nil {
			q.logger.ErrorWithFields("Failed to persist task status", map[string]interface{}{
				"task_id": update.taskID,
				"status":  update.status,
				"error":   err.Error(),
			})
		}
	}
}

// storeTask stores a task in the database
func (q *Queue) storeTask(task *WorkflowTask) error {
	// Skip database storage if database is not available
	if q.db == nil {
		return nil
	}

	workflowJSON, err := json.Marshal(task.Workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	metadataJSON, err := json.Marshal(task.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO queue_tasks (task_id, app_name, workflow_name, workflow_spec, metadata, status, enqueued_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = q.db.DB().Exec(query, task.ID, task.AppName, task.WorkflowName, workflowJSON, metadataJSON, TaskStatusPending, task.EnqueuedAt)
	if err != nil {
		return fmt.Errorf("failed to insert task: %w", err)
	}

	return nil
}

// persistTaskStatus updates task status in the database
func (q *Queue) persistTaskStatus(taskID string, status TaskStatus, taskErr error) error {
	// Skip database persistence if database is not available
	if q.db == nil {
		return nil
	}

	var errorMsg *string
	var completedAt *time.Time

	if taskErr != nil {
		msg := taskErr.Error()
		errorMsg = &msg
	}

	if status == TaskStatusCompleted || status == TaskStatusFailed {
		now := time.Now()
		completedAt = &now
	}

	query := `
		UPDATE queue_tasks
		SET status = $1, error_message = $2, completed_at = $3, updated_at = NOW()
		WHERE task_id = $4
	`

	_, err := q.db.DB().Exec(query, status, errorMsg, completedAt, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

// GetQueueStats returns queue statistics
func (q *Queue) GetQueueStats() map[string]interface{} {
	q.mu.RLock()
	activeCount := len(q.activeTasks)
	q.mu.RUnlock()

	stats := q.metricsCollector.getStats()
	stats["queue_size"] = len(q.tasks)
	stats["active_tasks"] = activeCount
	stats["workers"] = q.workers

	return stats
}

// GetActiveTasks returns currently executing tasks
func (q *Queue) GetActiveTasks() []*WorkflowTask {
	q.mu.RLock()
	defer q.mu.RUnlock()

	tasks := make([]*WorkflowTask, 0, len(q.activeTasks))
	for _, task := range q.activeTasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// generateTaskID generates a unique task ID
func generateTaskID() string {
	return fmt.Sprintf("task-%d-%d", time.Now().UnixNano(), time.Now().Unix()%1000)
}

// MetricsCollector methods

func (m *MetricsCollector) incrementEnqueued() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasksEnqueued++
}

func (m *MetricsCollector) recordTaskCompletion(queueTime, executionTime time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalQueueTime += queueTime
	m.totalExecutionTime += executionTime

	if success {
		m.tasksCompleted++
	} else {
		m.tasksFailed++
	}
}

func (m *MetricsCollector) getStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalTasks := m.tasksCompleted + m.tasksFailed
	var avgQueueTimeMs, avgExecutionTimeMs int64

	if totalTasks > 0 {
		avgQueueTimeMs = m.totalQueueTime.Milliseconds() / totalTasks
		avgExecutionTimeMs = m.totalExecutionTime.Milliseconds() / totalTasks
	}

	return map[string]interface{}{
		"tasks_enqueued":        m.tasksEnqueued,
		"tasks_completed":       m.tasksCompleted,
		"tasks_failed":          m.tasksFailed,
		"avg_queue_time_ms":     avgQueueTimeMs,
		"avg_execution_time_ms": avgExecutionTimeMs,
	}
}
