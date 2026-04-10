package retry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/go-github/v69/github"
)

const maxBackoffSeconds = 60

// sleepFn is the function used to sleep between retries.
// It is a package variable so tests can replace it with an instant channel.
var sleepFn = time.After

// Do calls fn up to maxAttempts times. It retries on GitHub rate limit errors
// and transient HTTP 5xx errors. All other errors are returned immediately.
// Every attempt, wait, and outcome is logged.
func Do(ctx context.Context, maxAttempts int, operation string, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		slog.Info("api call attempt", "operation", operation, "attempt", attempt, "max_attempts", maxAttempts)

		err := fn()
		if err == nil {
			slog.Info("api call succeeded", "operation", operation, "attempt", attempt)
			return nil
		}
		lastErr = err

		waitDur, shouldRetry := waitDuration(err, attempt)
		if !shouldRetry {
			slog.Error("api call failed, error is not retryable",
				"operation", operation,
				"attempt", attempt,
				"error", err,
			)
			return err
		}

		if attempt == maxAttempts {
			slog.Error("api call failed, max attempts reached",
				"operation", operation,
				"attempt", attempt,
				"error", err,
			)
			break
		}

		slog.Warn("api call failed, will retry",
			"operation", operation,
			"attempt", attempt,
			"max_attempts", maxAttempts,
			"wait_duration", waitDur.String(),
			"error", safeErrMsg(err),
		)

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting to retry %s: %w", operation, ctx.Err())
		case <-sleepFn(waitDur):
			slog.Info("retry wait complete, retrying", "operation", operation, "next_attempt", attempt+1)
		}
	}

	return fmt.Errorf("operation %s failed after %d attempts: %w", operation, maxAttempts, lastErr)
}

// waitDuration returns the duration to wait before the next attempt and
// whether the error is retryable. It handles GitHub rate limit errors,
// abuse rate limit errors, and transient 5xx HTTP errors.
func waitDuration(err error, attempt int) (time.Duration, bool) {
	var rateLimitErr *github.RateLimitError
	if errors.As(err, &rateLimitErr) {
		wait := time.Until(rateLimitErr.Rate.Reset.Time) + time.Second // +1s jitter
		if wait < time.Second {
			wait = time.Second
		}
		slog.Warn("github rate limit hit",
			"reset_at", rateLimitErr.Rate.Reset.Time,
			"wait", wait.String(),
		)
		return wait, true
	}

	var abuseErr *github.AbuseRateLimitError
	if errors.As(err, &abuseErr) {
		wait := abuseErr.GetRetryAfter()
		if wait <= 0 {
			wait = time.Second
		}
		slog.Warn("github abuse rate limit hit", "retry_after", wait.String())
		return wait, true
	}

	var errResp *github.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response != nil {
		if errResp.Response.StatusCode >= 500 {
			seconds := math.Min(math.Pow(2, float64(attempt-1)), float64(maxBackoffSeconds))
			wait := time.Duration(seconds) * time.Second
			slog.Warn("github transient error",
				"status_code", errResp.Response.StatusCode,
				"backoff", wait.String(),
			)
			return wait, true
		}
		return 0, false
	}

	return 0, false
}

// safeErrMsg calls err.Error() safely, catching panics from upstream Error()
// implementations that dereference nil fields (e.g. github.RateLimitError when
// Response is nil from the pre-flight rate-limit check).
func safeErrMsg(err error) string {
	var msg string
	func() {
		defer func() {
			if r := recover(); r != nil {
				msg = fmt.Sprintf("(error message unavailable: %v)", r)
			}
		}()
		msg = err.Error()
	}()
	return msg
}
