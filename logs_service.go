package main

import (
	"context"
	"log/slog"
	"sync"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
)

// extractAttr extracts the value of a given key from a slice of KeyValue pairs
func extractAttr(attributes []*commonpb.KeyValue, key string) (string, bool) {
	for _, attr := range attributes {
		if attr.Key == key {
			return attr.Value.GetStringValue(), true
		}
	}
	return "", false
}

type dash0LogsServiceServer struct {
	addr string
	// attributeKey is the key we're looking for in the attributes
	attributeKey string
	// counts stores the number of logs per attribute value
	counts map[string]int64
	mu     sync.RWMutex

	collogspb.UnimplementedLogsServiceServer
}

func newServer(addr string) collogspb.LogsServiceServer {
	s := &dash0LogsServiceServer{
		addr:         addr,
		attributeKey: "foo", // hardcoded attribute key for now
		counts:       make(map[string]int64),
	}
	return s
}

// process checks for the attribute key in the given attributes and updates the count if found
func (l *dash0LogsServiceServer) process(attributes []*commonpb.KeyValue, level string) {
	if value, found := extractAttr(attributes, l.attributeKey); found {
		l.mu.Lock()
		l.counts[value]++
		l.mu.Unlock()
		slog.Info("Found attribute", "level", level, "key", l.attributeKey, "value", value)
	}
}

func (l *dash0LogsServiceServer) Export(ctx context.Context, request *collogspb.ExportLogsServiceRequest) (*collogspb.ExportLogsServiceResponse, error) {
	slog.DebugContext(ctx, "Received ExportLogsServiceRequest")
	logsReceivedCounter.Add(ctx, 1)

	for _, resourceLog := range request.ResourceLogs {
		// check resource level attributes
		if resourceLog.Resource != nil {
			l.process(resourceLog.Resource.Attributes, "resource")
		}

		// Process each scope log
		for _, scopeLog := range resourceLog.ScopeLogs {
			// Check scope level attributes
			if scopeLog.Scope != nil {
				l.process(scopeLog.Scope.Attributes, "scope")
			}

			// Process each log record
			for _, logRecord := range scopeLog.LogRecords {
				l.process(logRecord.Attributes, "log")
			}
		}
	}

	return &collogspb.ExportLogsServiceResponse{}, nil
}
