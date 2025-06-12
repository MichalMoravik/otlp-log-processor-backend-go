package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	listenAddr            = flag.String("listenAddr", "localhost:4317", "The listen address")
	maxReceiveMessageSize = flag.Int("maxReceiveMessageSize", 16777216, "The max message size in bytes the server can receive")
)

const (
	name = "dash0.com/otlp-log-processor-backend"
	// Environment variable names
	envAttributeKey   = "ATTRIBUTE_KEY"
	envReportDuration = "REPORT_DURATION"
)

var (
	tracer              = otel.Tracer(name)
	meter               = otel.Meter(name)
	logger              = otelslog.NewLogger(name)
	logsReceivedCounter metric.Int64Counter
)

func init() {
	var err error
	logsReceivedCounter, err = meter.Int64Counter("com.dash0.homeexercise.logs.received",
		metric.WithDescription("The number of logs received by otlp-log-processor-backend"),
		metric.WithUnit("{log}"))
	if err != nil {
		panic(err)
	}
}

func getConfig() (string, time.Duration, error) {
	attributeKey := os.Getenv(envAttributeKey)
	if attributeKey == "" {
		return "", 0, fmt.Errorf("%s environment variable is required", envAttributeKey)
	}

	durationStr := os.Getenv(envReportDuration)
	if durationStr == "" {
		return "", 0, fmt.Errorf("%s environment variable is required", envReportDuration)
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid %s: %v", envReportDuration, err)
	}

	return attributeKey, duration, nil
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() (err error) {
	slog.SetDefault(logger)
	logger.Info("Starting application")

	// Set up OpenTelemetry.
	otelShutdown, err := setupOTelSDK(context.Background())
	if err != nil {
		return
	}

	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	flag.Parse()

	// Get configuration from environment variables
	attributeKey, reportDuration, err := getConfig()
	if err != nil {
		slog.Error("Failed to get config", "error", err)
		return
	}

	slog.Debug("Starting listener", slog.String("listenAddr", *listenAddr))
	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		return
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.MaxRecvMsgSize(*maxReceiveMessageSize),
		grpc.Creds(insecure.NewCredentials()),
	)

	server, err := newServer(*listenAddr, attributeKey, reportDuration)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		return err
	}
	collogspb.RegisterLogsServiceServer(grpcServer, server)

	slog.Debug("Starting gRPC server")
	if err := grpcServer.Serve(listener); err != nil {
		slog.Error("Failed to serve", "error", err)
		return err
	}
	return nil
}
