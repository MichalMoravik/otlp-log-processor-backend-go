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
	server := newServer("localhost:4317", "foo", 100*time.Millisecond).(*dash0LogsServiceServer)
	defer server.Stop() //clean up the background reporter

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

	// First export - should count to 1
	_, err := server.Export(context.Background(), request)
	require.NoError(t, err)

	// Wait for a short time to ensure counts are processed
	time.Sleep(10 * time.Millisecond)

	// Check counts after first export
	counts := server.getCounts()
	require.Equal(t, int64(1), counts["resource_value"])
	require.Equal(t, int64(1), counts["scope_value"])
	require.Equal(t, int64(1), counts["log_value"])

	// Wait for reporting interval
	time.Sleep(150 * time.Millisecond)

	// Check counts after reporting - should be 0
	counts = server.getCounts()
	require.Equal(t, int64(0), counts["resource_value"])
	require.Equal(t, int64(0), counts["scope_value"])
	require.Equal(t, int64(0), counts["log_value"])

	// Second export - should count to 1 again
	_, err = server.Export(context.Background(), request)
	require.NoError(t, err)

	// Wait for a short time to ensure counts are processed
	time.Sleep(10 * time.Millisecond)

	// Check counts after second export
	counts = server.getCounts()
	require.Equal(t, int64(1), counts["resource_value"])
	require.Equal(t, int64(1), counts["scope_value"])
	require.Equal(t, int64(1), counts["log_value"])
}
