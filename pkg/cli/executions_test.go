package cli

import (
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/stretchr/testify/assert"
)

func TestParseSinceFlag(t *testing.T) {
	tests := []struct {
		name      string
		since     string
		wantError bool
		checkFunc func(time.Time) bool
	}{
		{
			name:      "7 days",
			since:     "7d",
			wantError: false,
			checkFunc: func(result time.Time) bool {
				expected := time.Now().AddDate(0, 0, -7)
				return result.Before(time.Now()) && result.After(expected.Add(-1*time.Minute))
			},
		},
		{
			name:      "24 hours",
			since:     "24h",
			wantError: false,
			checkFunc: func(result time.Time) bool {
				expected := time.Now().Add(-24 * time.Hour)
				return result.Before(time.Now()) && result.After(expected.Add(-1*time.Minute))
			},
		},
		{
			name:      "date format",
			since:     "2025-01-05",
			wantError: false,
			checkFunc: func(result time.Time) bool {
				return result.Year() == 2025 && result.Month() == 1 && result.Day() == 5
			},
		},
		{
			name:      "invalid format",
			since:     "invalid",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSinceFlag(tt.since)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, tt.checkFunc(result), "Time check failed for %s", tt.since)
			}
		})
	}
}

func TestColorizeStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{
			name:   "completed",
			status: "completed",
			want:   colorGreen + "completed" + colorReset,
		},
		{
			name:   "failed",
			status: "failed",
			want:   colorRed + "failed" + colorReset,
		},
		{
			name:   "running",
			status: "running",
			want:   colorYellow + "running" + colorReset,
		},
		{
			name:   "pending",
			status: "pending",
			want:   colorGray + "pending" + colorReset,
		},
		{
			name:   "cancelled",
			status: "cancelled",
			want:   colorGray + "cancelled" + colorReset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := colorizeStatus(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetNodeSymbol(t *testing.T) {
	tests := []struct {
		name   string
		status execution.NodeStatus
		want   string
	}{
		{
			name:   "completed",
			status: execution.NodeStatusCompleted,
			want:   colorGreen + "✓" + colorReset,
		},
		{
			name:   "failed",
			status: execution.NodeStatusFailed,
			want:   colorRed + "✗" + colorReset,
		},
		{
			name:   "running",
			status: execution.NodeStatusRunning,
			want:   colorYellow + "●" + colorReset,
		},
		{
			name:   "skipped",
			status: execution.NodeStatusSkipped,
			want:   colorGray + "○" + colorReset,
		},
		{
			name:   "pending",
			status: execution.NodeStatusPending,
			want:   colorGray + "○" + colorReset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getNodeSymbol(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatDurationValue(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "zero",
			duration: 0,
			want:     "-",
		},
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			want:     "500ms",
		},
		{
			name:     "seconds",
			duration: 2300 * time.Millisecond,
			want:     "2.3s",
		},
		{
			name:     "minutes",
			duration: 90 * time.Second,
			want:     "1.5m",
		},
		{
			name:     "hours",
			duration: 2*time.Hour + 30*time.Minute,
			want:     "2.5h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDurationValue(tt.duration)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "no truncation needed",
			input:  "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "exact length",
			input:  "exactly10!",
			maxLen: 10,
			want:   "exactly10!",
		},
		{
			name:   "truncation needed",
			input:  "this is a very long string",
			maxLen: 10,
			want:   "this is..",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{
			name:  "nil",
			value: nil,
			want:  "null",
		},
		{
			name:  "string",
			value: "test",
			want:  `"test"`,
		},
		{
			name:  "long string",
			value: "this is a very long string that should be truncated because it exceeds the maximum display length limit",
			want:  `"this is a very long string that should be truncated because it exceeds the maximum display len...`,
		},
		{
			name:  "bool",
			value: true,
			want:  "true",
		},
		{
			name:  "int",
			value: 42,
			want:  "42",
		},
		{
			name:  "float",
			value: 3.14,
			want:  "3.14",
		},
		{
			name:  "map",
			value: map[string]interface{}{"key": "value"},
			want:  `{"key":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}
