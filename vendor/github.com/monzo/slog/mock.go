package slog

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockLogger struct {
	mock.Mock
}

var _ LeveledLogger = &MockLogger{}

func (m *MockLogger) Critical(ctx context.Context, msg string, params ...interface{}) {
	m.Called(ctx, msg, params)
}

func (m *MockLogger) Error(ctx context.Context, msg string, params ...interface{}) {
	m.Called(ctx, msg, params)
}

func (m *MockLogger) Warn(ctx context.Context, msg string, params ...interface{}) {
	m.Called(ctx, msg, params)
}

func (m *MockLogger) Info(ctx context.Context, msg string, params ...interface{}) {
	m.Called(ctx, msg, params)
}

func (m *MockLogger) Debug(ctx context.Context, msg string, params ...interface{}) {
	m.Called(ctx, msg, params)
}

func (m *MockLogger) Trace(ctx context.Context, msg string, params ...interface{}) {
	m.Called(ctx, msg, params)
}
