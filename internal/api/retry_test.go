package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientWithRetry(t *testing.T) {
	tests := []struct {
		name           string
		serverBehavior func(attemptCount *int) http.HandlerFunc
		maxRetries     int
		expectError    bool
		expectAttempts int
	}{
		{
			name: "succeeds on first attempt",
			serverBehavior: func(attemptCount *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*attemptCount++
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"roles": []}`))
				}
			},
			maxRetries:     3,
			expectError:    false,
			expectAttempts: 1,
		},
		{
			name: "succeeds on third attempt",
			serverBehavior: func(attemptCount *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*attemptCount++
					if *attemptCount < 3 {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"roles": []}`))
				}
			},
			maxRetries:     3,
			expectError:    false,
			expectAttempts: 3,
		},
		{
			name: "fails after max retries",
			serverBehavior: func(attemptCount *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*attemptCount++
					w.WriteHeader(http.StatusInternalServerError)
				}
			},
			maxRetries:     2,
			expectError:    true,
			expectAttempts: 3, // initial + 2 retries
		},
		{
			name: "respects context cancellation",
			serverBehavior: func(attemptCount *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*attemptCount++
					time.Sleep(100 * time.Millisecond) // Simulate slow response
					w.WriteHeader(http.StatusInternalServerError)
				}
			},
			maxRetries:     5,
			expectError:    true,
			expectAttempts: 1, // Should stop early due to context cancellation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attemptCount := 0
			server := httptest.NewServer(tt.serverBehavior(&attemptCount))
			defer server.Close()

			logger := createTestLogger()
			client, err := NewClientWithRetry(server.URL, "test-token", logger, tt.maxRetries)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			ctx := context.Background()
			if tt.name == "respects context cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer cancel()
			}

			_, err = client.GetRolesWithContext(ctx)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if attemptCount != tt.expectAttempts {
				t.Errorf("Expected %d attempts, got %d", tt.expectAttempts, attemptCount)
			}
		})
	}
}

func TestRetryBackoff(t *testing.T) {
	attemptCount := 0
	startTime := time.Now()
	var attemptTimes []time.Time

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		attemptTimes = append(attemptTimes, time.Now())
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := createTestLogger()
	client, err := NewClientWithRetry(server.URL, "test-token", logger, 3)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	_, err = client.GetRolesWithContext(ctx)

	if err == nil {
		t.Error("Expected error but got none")
	}

	if attemptCount != 4 { // initial + 3 retries
		t.Errorf("Expected 4 attempts, got %d", attemptCount)
	}

	// Verify exponential backoff timing (allow some margin for test execution)
	if len(attemptTimes) >= 2 {
		firstDelay := attemptTimes[1].Sub(attemptTimes[0])
		if firstDelay < 900*time.Millisecond {
			t.Errorf("First retry delay too short: %v", firstDelay)
		}
	}

	if len(attemptTimes) >= 3 {
		secondDelay := attemptTimes[2].Sub(attemptTimes[1])
		if secondDelay < 1800*time.Millisecond { // 2^1 = 2s, allow 10% margin
			t.Errorf("Second retry delay too short: %v", secondDelay)
		}
	}

	totalTime := time.Since(startTime)
	if totalTime < 6*time.Second { // 1s + 2s + 4s = 7s, allow some margin
		t.Errorf("Total retry time too short: %v", totalTime)
	}
}
