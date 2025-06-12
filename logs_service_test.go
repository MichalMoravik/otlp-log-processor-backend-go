package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	otellogs "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
)

func TestExtractAttr(t *testing.T) {
	attributes := []*commonpb.KeyValue{
		{
			Key: "foo",
			Value: &commonpb.AnyValue{
				Value: &commonpb.AnyValue_StringValue{
					StringValue: "bar",
				},
			},
		},
		{
			Key: "baz",
			Value: &commonpb.AnyValue{
				Value: &commonpb.AnyValue_StringValue{
					StringValue: "qux",
				},
			},
		},
	}

	tests := []struct {
		name     string
		key      string
		expected string
		found    bool
	}{
		{
			name:     "existing key",
			key:      "foo",
			expected: "bar",
			found:    true,
		},
		{
			name:     "non-existing key",
			key:      "nonexistent",
			expected: "",
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := extractAttr(attributes, tt.key)
			assert.Equal(t, tt.expected, value)
			assert.Equal(t, tt.found, found)
		})
	}
}

func TestExport_CountingAndReporting(t *testing.T) {
	server := &dash0LogsServiceServer{
		attributeKey:   "foo",
		counts:         make(map[string]int64),
		reportDuration: 100 * time.Millisecond, // Short duration for testing
		lastReport:     time.Now(),
	}

	request := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*otellogs.ResourceLogs{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{
							Key: "foo",
							Value: &commonpb.AnyValue{
								Value: &commonpb.AnyValue_StringValue{
									StringValue: "resource_value",
								},
							},
						},
					},
				},
				ScopeLogs: []*otellogs.ScopeLogs{
					{
						Scope: &commonpb.InstrumentationScope{
							Attributes: []*commonpb.KeyValue{
								{
									Key: "foo",
									Value: &commonpb.AnyValue{
										Value: &commonpb.AnyValue_StringValue{
											StringValue: "scope_value",
										},
									},
								},
							},
						},
						LogRecords: []*otellogs.LogRecord{
							{
								Attributes: []*commonpb.KeyValue{
									{
										Key: "foo",
										Value: &commonpb.AnyValue{
											Value: &commonpb.AnyValue_StringValue{
												StringValue: "log_value",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Test 1: Verify counting works
	_, err := server.Export(context.Background(), request)
	require.NoError(t, err)

	// Check counts immediately after processing
	counts := server.getCounts()
	assert.Equal(t, int64(1), counts["resource_value"])
	assert.Equal(t, int64(1), counts["scope_value"])
	assert.Equal(t, int64(1), counts["log_value"])

	// Test 2: Verify reporting happens after duration
	time.Sleep(150 * time.Millisecond) // Wait for reporting duration

	// Send another request to trigger reporting
	_, err = server.Export(context.Background(), request)
	require.NoError(t, err)

	// Check that counts are 1 after reporting
	counts = server.getCounts()
	assert.Equal(t, int64(1), counts["resource_value"])
	assert.Equal(t, int64(1), counts["scope_value"])
	assert.Equal(t, int64(1), counts["log_value"])

	// Test 3: Send another batch, counts should be 2
	_, err = server.Export(context.Background(), request)
	require.NoError(t, err)
	counts = server.getCounts()
	assert.Equal(t, int64(2), counts["resource_value"])
	assert.Equal(t, int64(2), counts["scope_value"])
	assert.Equal(t, int64(2), counts["log_value"])
}
