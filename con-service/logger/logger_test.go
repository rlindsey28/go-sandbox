package logger

import (
	"context"
	"testing"
)

func TestGet(t *testing.T) {
	logger := Get()
	if logger == nil {
		t.Error("Expected logger to be initialized, but it was nil")
	}
}

func TestFromCtx(t *testing.T) {
	ctx := context.Background()
	logger := FromCtx(ctx)
	if logger == nil {
		t.Error("Expected logger to be initialized, but it was nil")
	}
}

func TestWithCtx(t *testing.T) {
	ctx := context.Background()
	logger := Get()
	ctxWithLogger := WithCtx(ctx, logger)

	if ctxWithLogger.Value(ctxKey{}) != logger {
		t.Error("Expected logger to be attached to context, but it was not")
	}
}
