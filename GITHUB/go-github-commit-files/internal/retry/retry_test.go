package retry

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/go-github/v69/github"
)

// TestMain replaces sleepFn with an instant channel so retry tests don't
// actually wait seconds for backoff delays.
func TestMain(m *testing.M) {
	sleepFn = func(d time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}
	os.Exit(m.Run())
}

func TestDo_successOnFirstCall(t *testing.T) {
	calls := 0
	err := Do(context.Background(), 3, "test_op", func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("want 1 call, got %d", calls)
	}
}

func TestDo_successAfterRetry(t *testing.T) {
	calls := 0
	err := Do(context.Background(), 3, "test_op", func() error {
		calls++
		if calls < 3 {
			return &github.ErrorResponse{
				Response: &http.Response{StatusCode: 503},
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("want 3 calls, got %d", calls)
	}
}

func TestDo_exhaustsRetries(t *testing.T) {
	calls := 0
	err := Do(context.Background(), 3, "test_op", func() error {
		calls++
		return &github.ErrorResponse{
			Response: &http.Response{StatusCode: 500},
		}
	})
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
	if calls != 3 {
		t.Errorf("want 3 calls, got %d", calls)
	}
}

func TestDo_nonRetryableErrorReturnedImmediately(t *testing.T) {
	calls := 0
	sentinel := errors.New("not retryable")
	err := Do(context.Background(), 5, "test_op", func() error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("want sentinel error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("want 1 call (immediate return), got %d", calls)
	}
}

func TestDo_rateLimitErrorRetried(t *testing.T) {
	calls := 0
	pastReset := github.Timestamp{Time: time.Now().Add(-5 * time.Second)}
	err := Do(context.Background(), 3, "test_op", func() error {
		calls++
		if calls == 1 {
			return &github.RateLimitError{
				Rate:    github.Rate{Reset: pastReset},
				Message: "rate limit exceeded",
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("want 2 calls, got %d", calls)
	}
}

func TestDo_contextCancelledDuringWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	err := Do(ctx, 5, "test_op", func() error {
		calls++
		cancel()
		return &github.ErrorResponse{
			Response: &http.Response{StatusCode: 500},
		}
	})
	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}
}

func TestWaitDuration_rateLimitError(t *testing.T) {
	resetAt := time.Now().Add(10 * time.Second)
	err := &github.RateLimitError{
		Rate: github.Rate{Reset: github.Timestamp{Time: resetAt}},
	}
	dur, retry := waitDuration(err, 1)
	if !retry {
		t.Error("expected retry=true for rate limit error")
	}
	if dur < 9*time.Second || dur > 12*time.Second {
		t.Errorf("unexpected wait duration %v, want ~11s", dur)
	}
}

func TestWaitDuration_5xxError(t *testing.T) {
	err := &github.ErrorResponse{
		Response: &http.Response{StatusCode: 500},
	}
	dur, retry := waitDuration(err, 1)
	if !retry {
		t.Error("expected retry=true for 5xx error")
	}
	if dur != 1*time.Second {
		t.Errorf("want 1s backoff on attempt 1, got %v", dur)
	}

	dur2, _ := waitDuration(err, 2)
	if dur2 != 2*time.Second {
		t.Errorf("want 2s backoff on attempt 2, got %v", dur2)
	}

	dur3, _ := waitDuration(err, 3)
	if dur3 != 4*time.Second {
		t.Errorf("want 4s backoff on attempt 3, got %v", dur3)
	}
}

func TestWaitDuration_4xxError_notRetried(t *testing.T) {
	err := &github.ErrorResponse{
		Response: &http.Response{StatusCode: 404},
	}
	_, retry := waitDuration(err, 1)
	if retry {
		t.Error("expected retry=false for 4xx error")
	}
}
