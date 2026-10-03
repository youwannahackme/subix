package utils

import (
	"context"
	"testing"
)

func TestRateLimiterWait(t *testing.T) {
	rate := 5 // 5 req/s
	rl := NewRateLimiter(rate)
	if rl == nil {
		t.Fatal("expected non-nil rate limiter")
	}

	// Consume burst tokens
	for i := 0; i < rate; i++ {
		if !rl.TryWait() {
			t.Fatalf("expected TryWait() to succeed for initial token %d", i)
		}
	}

	// Next TryWait should fail
	if rl.TryWait() {
		t.Fatal("expected TryWait() to fail after exhausting tokens")
	}
}

func TestRateLimiterWaitContextCancellation(t *testing.T) {
	rate := 1
	rl := NewRateLimiter(rate)

	// Consume the single token
	rl.Wait()

	// Now try to wait with a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := rl.WaitContext(ctx)
	if err == nil {
		t.Fatal("expected WaitContext to return error on cancelled context")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled error, got: %v", err)
	}
}
