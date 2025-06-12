package main

import (
	"context"
	"testing"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	otellogs "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
)

func TestExtractAttr(t *testing.T) {
	tests := []struct {
		name       string
		attributes []*commonpb.KeyValue
		key        string
		wantValue  string
		wantFound  bool
	}{
		{
			name: "attribute found",
			attributes: []*commonpb.KeyValue{
				{Key: "foo", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "bar"}}},
			},
			key:       "foo",
			wantValue: "bar",
			wantFound: true,
		},
		{
			name: "attribute not found",
			attributes: []*commonpb.KeyValue{
				{Key: "baz", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "qux"}}},
			},
			key:       "foo",
			wantValue: "",
			wantFound: false,
		},
		{
			name:       "empty attributes",
			attributes: []*commonpb.KeyValue{},
			key:        "foo",
			wantValue:  "",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotFound := extractAttr(tt.attributes, tt.key)
			if gotValue != tt.wantValue || gotFound != tt.wantFound {
				t.Errorf("extractAttr() = (%v, %v), want (%v, %v)",
					gotValue, gotFound, tt.wantValue, tt.wantFound)
			}
		})
	}
}

func TestExport_AllLevels(t *testing.T) {
	server := newServer("localhost:4317")

	// Create a test request with attributes at all levels
	request := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*otellogs.ResourceLogs{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{Key: "foo", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "resource_value"}}},
					},
				},
				ScopeLogs: []*otellogs.ScopeLogs{
					{
						Scope: &commonpb.InstrumentationScope{
							Attributes: []*commonpb.KeyValue{
								{Key: "foo", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "scope_value"}}},
							},
						},
						LogRecords: []*otellogs.LogRecord{
							{
								Attributes: []*commonpb.KeyValue{
									{Key: "foo", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "log_value"}}},
								},
							},
						},
					},
				},
			},
		},
	}

	// Call Export
	_, err := server.Export(context.Background(), request)
	if err != nil {
		t.Errorf("Export() error = %v", err)
	}
}
