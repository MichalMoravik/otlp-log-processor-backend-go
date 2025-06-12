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

func (l *dash0LogsServiceServer) Export(ctx context.Context, request *collogspb.ExportLogsServiceRequest) (*collogspb.ExportLogsServiceResponse, error) {
	slog.DebugContext(ctx, "Received ExportLogsServiceRequest")
	logsReceivedCounter.Add(ctx, 1)

	for _, resourceLog := range request.ResourceLogs {
		// check resource level attributes
		if resourceLog.Resource != nil {
			if value, found := extractAttr(resourceLog.Resource.Attributes, l.attributeKey); found {
				l.mu.Lock()
				l.counts[value]++
				l.mu.Unlock()
				slog.Info("Found attribute at resource level", "key", l.attributeKey, "value", value)
			}
		}

		for _, scopeLog := range resourceLog.ScopeLogs {
			// Check scope level attributes
			if scopeLog.Scope != nil {
				if value, found := extractAttr(scopeLog.Scope.Attributes, l.attributeKey); found {
					l.mu.Lock()
					l.counts[value]++
					l.mu.Unlock()
					slog.Info("Found attribute at scope level", "key", l.attributeKey, "value", value)
				}
			}

			for _, logRecord := range scopeLog.LogRecords {
				// Check log level attributes
				if value, found := extractAttr(logRecord.Attributes, l.attributeKey); found {
					l.mu.Lock()
					l.counts[value]++
					l.mu.Unlock()
					slog.Info("Found attribute at log level", "key", l.attributeKey, "value", value)
				}
			}
		}
	}

	return &collogspb.ExportLogsServiceResponse{}, nil
}
