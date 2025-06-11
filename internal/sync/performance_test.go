package sync

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"replbac/internal/models"
)

// TestConcurrentSyncPerformance tests that sync operations can be executed concurrently
func TestConcurrentSyncPerformance(t *testing.T) {
	tests := []struct {
		name                    string
		numRoles                int
		concurrency             int
		expectedMaxDuration     time.Duration
		expectedMinConcurrency  int
	}{
		{
			name:                   "small batch with concurrency",
			numRoles:               5,
			concurrency:            3,
			expectedMaxDuration:    time.Second * 2,
			expectedMinConcurrency: 3,
		},
		{
			name:                   "medium batch with higher concurrency",
			numRoles:               20,
			concurrency:            10,
			expectedMaxDuration:    time.Second * 3,
			expectedMinConcurrency: 10,
		},
		{
			name:                   "large batch with controlled concurrency",
			numRoles:               50,
			concurrency:            5,
			expectedMaxDuration:    time.Second * 10,
			expectedMinConcurrency: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client that tracks concurrent operations
			mockClient := &ConcurrentTrackingClient{
				operationDelay:     time.Millisecond * 100, // Simulate API latency
				maxConcurrentCalls: 0,
				currentCalls:       0,
				mu:                 sync.Mutex{},
			}

			// Create executor with concurrency settings
			executor := NewConcurrentExecutor(mockClient, nil, tt.concurrency)

			// Generate test plan with many creates
			plan := SyncPlan{
				Creates: generateTestRoles(tt.numRoles),
			}

			// Execute with timing
			start := time.Now()
			result := executor.ExecutePlan(plan)
			duration := time.Since(start)

			// Verify successful execution
			if result.Error != nil {
				t.Fatalf("Expected successful execution, got error: %v", result.Error)
			}

			// Verify performance improvements
			if duration > tt.expectedMaxDuration {
				t.Errorf("Execution took too long: %v > %v", duration, tt.expectedMaxDuration)
			}

			// Verify concurrency was actually used
			if mockClient.maxConcurrentCalls < tt.expectedMinConcurrency {
				t.Errorf("Expected at least %d concurrent calls, got %d", 
					tt.expectedMinConcurrency, mockClient.maxConcurrentCalls)
			}

			// Verify all operations completed
			if result.Created != tt.numRoles {
				t.Errorf("Expected %d creates, got %d", tt.numRoles, result.Created)
			}
		})
	}
}

// TestConcurrentExecutorErrorHandling tests error handling during concurrent operations
func TestConcurrentExecutorErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		numRoles       int
		numFailures    int
		concurrency    int
		expectRollback bool
	}{
		{
			name:           "partial failure with rollback",
			numRoles:       10,
			numFailures:    3,
			concurrency:    5,
			expectRollback: true,
		},
		{
			name:           "single failure stops execution",
			numRoles:       20,
			numFailures:    1,
			concurrency:    10,
			expectRollback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client that fails some operations
			mockClient := &FailingClient{
				failureCount: tt.numFailures,
				callCount:    0,
				mu:           sync.Mutex{},
			}

			executor := NewConcurrentExecutor(mockClient, nil, tt.concurrency)
			
			plan := SyncPlan{
				Creates: generateTestRoles(tt.numRoles),
			}

			result := executor.ExecutePlan(plan)

			// Should have failed
			if result.Error == nil {
				t.Errorf("Expected execution to fail, but it succeeded")
			}

			// Should have performed rollback if expected
			if tt.expectRollback && mockClient.deleteCount == 0 {
				t.Errorf("Expected rollback operations, but none were performed")
			}

			// Should not have created more roles than succeeded before failure
			if result.Created >= tt.numRoles {
				t.Errorf("Created too many roles despite failures: %d >= %d", 
					result.Created, tt.numRoles)
			}
		})
	}
}

// TestConcurrentExecutorResourceLimits tests that concurrency limits are respected
func TestConcurrentExecutorResourceLimits(t *testing.T) {
	// Create executor with strict concurrency limit
	mockClient := &ConcurrentTrackingClient{
		operationDelay: time.Millisecond * 50,
		mu:             sync.Mutex{},
	}

	executor := NewConcurrentExecutor(mockClient, nil, 3) // Max 3 concurrent ops

	plan := SyncPlan{
		Creates: generateTestRoles(20), // Many operations
	}

	start := time.Now()
	result := executor.ExecutePlan(plan)
	duration := time.Since(start)

	if result.Error != nil {
		t.Fatalf("Expected successful execution, got error: %v", result.Error)
	}

	// Verify concurrency limit was respected
	if mockClient.maxConcurrentCalls > 3 {
		t.Errorf("Concurrency limit violated: %d > 3", mockClient.maxConcurrentCalls)
	}

	// Should still complete in reasonable time due to concurrency
	maxExpectedDuration := time.Second * 5 // 20 ops * 50ms / 3 concurrent + overhead
	if duration > maxExpectedDuration {
		t.Errorf("Execution took too long even with concurrency: %v > %v", 
			duration, maxExpectedDuration)
	}
}

// TestProgressTrackingDuringConcurrentOps tests progress reporting during concurrent execution
func TestProgressTrackingDuringConcurrentOps(t *testing.T) {
	mockClient := &ConcurrentTrackingClient{
		operationDelay: time.Millisecond * 100,
		mu:             sync.Mutex{},
	}

	progressReports := make([]string, 0)
	progressMu := sync.Mutex{}
	
	mockLogger := &ProgressTrackingLogger{
		onProgress: func(message string) {
			progressMu.Lock()
			progressReports = append(progressReports, message)
			progressMu.Unlock()
		},
	}

	executor := NewConcurrentExecutor(mockClient, mockLogger, 5)
	
	plan := SyncPlan{
		Creates: generateTestRoles(10),
		Updates: generateTestUpdateRoles(5),
		Deletes: []string{"old1", "old2", "old3"},
	}

	result := executor.ExecutePlan(plan)

	if result.Error != nil {
		t.Fatalf("Expected successful execution, got error: %v", result.Error)
	}

	// Should have received progress reports
	progressMu.Lock()
	numReports := len(progressReports)
	progressMu.Unlock()

	if numReports == 0 {
		t.Errorf("Expected progress reports during execution, got none")
	}

	// Should report progress for different operation types
	hasCreateProgress := false
	hasUpdateProgress := false
	hasDeleteProgress := false

	progressMu.Lock()
	for _, report := range progressReports {
		if containsString(report, "create") || containsString(report, "creating") {
			hasCreateProgress = true
		}
		if containsString(report, "update") || containsString(report, "updating") {
			hasUpdateProgress = true
		}
		if containsString(report, "delete") || containsString(report, "deleting") {
			hasDeleteProgress = true
		}
	}
	progressMu.Unlock()

	if !hasCreateProgress {
		t.Errorf("Expected create progress reports")
	}
	if !hasUpdateProgress {
		t.Errorf("Expected update progress reports")
	}
	if !hasDeleteProgress {
		t.Errorf("Expected delete progress reports")
	}
}

// Helper functions and mock types

func generateTestRoles(count int) []models.Role {
	roles := make([]models.Role, count)
	for i := 0; i < count; i++ {
		roles[i] = models.Role{
			Name: fmt.Sprintf("test-role-%d", i),
			Resources: models.Resources{
				Allowed: []string{"read", "write"},
				Denied:  []string{},
			},
		}
	}
	return roles
}

func generateTestUpdateRoles(count int) []models.Role {
	roles := make([]models.Role, count)
	for i := 0; i < count; i++ {
		roles[i] = models.Role{
			Name: fmt.Sprintf("update-role-%d", i),
			Resources: models.Resources{
				Allowed: []string{"read"},
				Denied:  []string{"write"},
			},
		}
	}
	return roles
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		   len(s) > len(substr) && s[:len(substr)] == substr ||
		   len(s) > len(substr) && s[len(s)/2-len(substr)/2:len(s)/2+len(substr)/2+len(substr)%2] == substr
}

// Mock client that tracks concurrent operations
type ConcurrentTrackingClient struct {
	operationDelay     time.Duration
	maxConcurrentCalls int
	currentCalls       int
	mu                 sync.Mutex
}

func (c *ConcurrentTrackingClient) trackCall() func() {
	c.mu.Lock()
	c.currentCalls++
	if c.currentCalls > c.maxConcurrentCalls {
		c.maxConcurrentCalls = c.currentCalls
	}
	c.mu.Unlock()

	return func() {
		c.mu.Lock()
		c.currentCalls--
		c.mu.Unlock()
	}
}

func (c *ConcurrentTrackingClient) GetRoles() ([]models.Role, error) {
	defer c.trackCall()()
	time.Sleep(c.operationDelay)
	return []models.Role{}, nil
}

func (c *ConcurrentTrackingClient) GetRole(roleName string) (models.Role, error) {
	defer c.trackCall()()
	time.Sleep(c.operationDelay)
	return models.Role{}, fmt.Errorf("role not found: %s", roleName)
}

func (c *ConcurrentTrackingClient) CreateRole(role models.Role) error {
	defer c.trackCall()()
	time.Sleep(c.operationDelay)
	return nil
}

func (c *ConcurrentTrackingClient) UpdateRole(role models.Role) error {
	defer c.trackCall()()
	time.Sleep(c.operationDelay)
	return nil
}

func (c *ConcurrentTrackingClient) DeleteRole(roleName string) error {
	defer c.trackCall()()
	time.Sleep(c.operationDelay)
	return nil
}

// Mock client that fails some operations
type FailingClient struct {
	failureCount int
	callCount    int
	deleteCount  int
	mu           sync.Mutex
}

func (c *FailingClient) GetRoles() ([]models.Role, error) {
	return []models.Role{}, nil
}

func (c *FailingClient) GetRole(roleName string) (models.Role, error) {
	return models.Role{}, fmt.Errorf("role not found: %s", roleName)
}

func (c *FailingClient) CreateRole(role models.Role) error {
	c.mu.Lock()
	c.callCount++
	shouldFail := c.callCount <= c.failureCount
	c.mu.Unlock()

	if shouldFail {
		return fmt.Errorf("simulated API failure for role: %s", role.Name)
	}
	return nil
}

func (c *FailingClient) UpdateRole(role models.Role) error {
	c.mu.Lock()
	c.callCount++
	shouldFail := c.callCount <= c.failureCount
	c.mu.Unlock()

	if shouldFail {
		return fmt.Errorf("simulated API failure for role: %s", role.Name)
	}
	return nil
}

func (c *FailingClient) DeleteRole(roleName string) error {
	c.mu.Lock()
	c.deleteCount++
	c.mu.Unlock()
	return nil
}

// Mock logger for progress tracking
type ProgressTrackingLogger struct {
	onProgress func(string)
}

func (l *ProgressTrackingLogger) Debug(format string, args ...interface{}) {
	if l.onProgress != nil {
		l.onProgress(fmt.Sprintf(format, args...))
	}
}

func (l *ProgressTrackingLogger) Info(format string, args ...interface{}) {
	if l.onProgress != nil {
		l.onProgress(fmt.Sprintf(format, args...))
	}
}

func (l *ProgressTrackingLogger) Warn(format string, args ...interface{}) {
	if l.onProgress != nil {
		l.onProgress(fmt.Sprintf(format, args...))
	}
}

func (l *ProgressTrackingLogger) Error(format string, args ...interface{}) {
	if l.onProgress != nil {
		l.onProgress(fmt.Sprintf(format, args...))
	}
}

func (l *ProgressTrackingLogger) TimedOperation(name string, operation func() error) error {
	if l.onProgress != nil {
		l.onProgress(fmt.Sprintf("starting %s", name))
	}
	err := operation()
	if l.onProgress != nil {
		if err != nil {
			l.onProgress(fmt.Sprintf("completed %s with error", name))
		} else {
			l.onProgress(fmt.Sprintf("completed %s successfully", name))
		}
	}
	return err
}