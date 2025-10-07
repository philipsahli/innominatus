package queue

import (
	"innominatus/internal/types"
	"sync"
	"testing"
	"time"
)

// MockExecutor implements WorkflowExecutor for testing
type MockExecutor struct {
	mu         sync.Mutex
	executions []string
	shouldFail bool
}

func (m *MockExecutor) ExecuteWorkflowWithName(appName, workflowName string, workflow types.Workflow) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executions = append(m.executions, appName+":"+workflowName)
	if m.shouldFail {
		return &ErrWorkflowExecutionFailed{AppName: appName, WorkflowName: workflowName}
	}
	return nil
}

func (m *MockExecutor) getExecutions() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.executions))
	copy(result, m.executions)
	return result
}

type ErrWorkflowExecutionFailed struct {
	AppName      string
	WorkflowName string
}

func (e *ErrWorkflowExecutionFailed) Error() string {
	return "workflow execution failed"
}

func TestQueue_EnqueueAndProcess(t *testing.T) {
	executor := &MockExecutor{}
	q := NewQueue(2, executor, nil)
	q.Start()
	defer q.Stop()

	workflow := types.Workflow{
		Steps: []types.Step{
			{Name: "test-step", Type: "dummy"},
		},
	}

	// Enqueue a task
	taskID, err := q.Enqueue("test-app", "test-workflow", workflow, nil)
	if err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}

	if taskID == "" {
		t.Fatal("Task ID should not be empty")
	}

	// Wait for task to be processed
	time.Sleep(1 * time.Second)

	// Verify task was executed
	executions := executor.getExecutions()
	if len(executions) != 1 {
		t.Errorf("Expected 1 execution, got %d", len(executions))
	}

	expected := "test-app:test-workflow"
	if len(executions) > 0 && executions[0] != expected {
		t.Errorf("Expected execution '%s', got '%s'", expected, executions[0])
	}
}

func TestQueue_MultipleWorkers(t *testing.T) {
	executor := &MockExecutor{}
	q := NewQueue(3, executor, nil)
	q.Start()
	defer q.Stop()

	workflow := types.Workflow{
		Steps: []types.Step{
			{Name: "test-step", Type: "dummy"},
		},
	}

	// Enqueue multiple tasks
	taskCount := 5
	for i := 0; i < taskCount; i++ {
		_, err := q.Enqueue("test-app", "test-workflow", workflow, nil)
		if err != nil {
			t.Fatalf("Failed to enqueue task %d: %v", i, err)
		}
	}

	// Wait for all tasks to be processed
	time.Sleep(2 * time.Second)

	// Verify all tasks were executed
	executions := executor.getExecutions()
	if len(executions) != taskCount {
		t.Errorf("Expected %d executions, got %d", taskCount, len(executions))
	}
}

func TestQueue_GetQueueStats(t *testing.T) {
	executor := &MockExecutor{}
	q := NewQueue(2, executor, nil)
	q.Start()
	defer q.Stop()

	workflow := types.Workflow{
		Steps: []types.Step{
			{Name: "test-step", Type: "dummy"},
		},
	}

	// Enqueue a task
	_, err := q.Enqueue("test-app", "test-workflow", workflow, nil)
	if err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}

	stats := q.GetQueueStats()

	if stats["workers"] != 2 {
		t.Errorf("Expected 2 workers, got %v", stats["workers"])
	}

	if stats["tasks_enqueued"].(int64) != 1 {
		t.Errorf("Expected 1 enqueued task, got %v", stats["tasks_enqueued"])
	}

	// Wait for task to complete
	time.Sleep(1 * time.Second)

	stats = q.GetQueueStats()
	if stats["tasks_completed"].(int64) != 1 {
		t.Errorf("Expected 1 completed task, got %v", stats["tasks_completed"])
	}
}

func TestQueue_FailedExecution(t *testing.T) {
	executor := &MockExecutor{shouldFail: true}
	q := NewQueue(1, executor, nil)
	q.Start()
	defer q.Stop()

	workflow := types.Workflow{
		Steps: []types.Step{
			{Name: "test-step", Type: "dummy"},
		},
	}

	// Enqueue a task that will fail
	_, err := q.Enqueue("test-app", "test-workflow", workflow, nil)
	if err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}

	// Wait for task to fail
	time.Sleep(1 * time.Second)

	stats := q.GetQueueStats()
	if stats["tasks_failed"].(int64) != 1 {
		t.Errorf("Expected 1 failed task, got %v", stats["tasks_failed"])
	}
}

func TestQueue_GetActiveTasks(t *testing.T) {
	// Create executor that sleeps to keep task active
	executor := &MockExecutor{}
	q := NewQueue(2, executor, nil)
	q.Start()
	defer q.Stop()

	workflow := types.Workflow{
		Steps: []types.Step{
			{Name: "test-step", Type: "dummy"},
		},
	}

	// Enqueue tasks
	_, err := q.Enqueue("app1", "workflow1", workflow, nil)
	if err != nil {
		t.Fatalf("Failed to enqueue task 1: %v", err)
	}

	_, err = q.Enqueue("app2", "workflow2", workflow, nil)
	if err != nil {
		t.Fatalf("Failed to enqueue task 2: %v", err)
	}

	// Give workers time to pick up tasks
	time.Sleep(100 * time.Millisecond)

	activeTasks := q.GetActiveTasks()

	// Should have some active tasks (might be 0, 1, or 2 depending on execution speed)
	// We just verify the method works without panicking
	if activeTasks == nil {
		t.Error("GetActiveTasks returned nil")
	}
}

func TestQueue_StopGracefully(t *testing.T) {
	executor := &MockExecutor{}
	q := NewQueue(2, executor, nil)
	q.Start()

	workflow := types.Workflow{
		Steps: []types.Step{
			{Name: "test-step", Type: "dummy"},
		},
	}

	// Enqueue a task
	_, err := q.Enqueue("test-app", "test-workflow", workflow, nil)
	if err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}

	// Give worker time to pick up the task
	time.Sleep(200 * time.Millisecond)

	// Stop queue (should wait for workers to finish)
	q.Stop()

	// Verify task was executed before shutdown
	executions := executor.getExecutions()
	if len(executions) != 1 {
		t.Errorf("Expected 1 execution before shutdown, got %d", len(executions))
	}
}
