# Processor

A log processor made in Golang. My simple solution of counting the attribute key values in logs can be found in `logs_service.go`.

## Running

The service is Dockerized so that it's easily runnable on any machine. To run it:

1. Run `docker compose up` (or for compose v1: `docker-compose up`) in the root of the project
2. The code is then compiled during the build step and the executable should be running

## Testing

To run the tests:

1. Make sure your Go version is 1.23
2. In the root of the project run: `go mod tidy`
3. Run `go test ./...`

**TODO:**
Ideally, tests should be run within a separate Docker container, even better as a part of a CI/CD pipeline. This could be one of the next steps in development.

## Reporting Duration

Reporting is run within a separate go routine, independent of the Export function. It's stopped and closed when the service stops running.

## Concurrency

I could see the potential for using a workers pool (semaphore) but in this case, since we use the Lock before counting, the Lock would become a bottleneck anyway. If this was a different operation than counting of occurrences, e.g. publishing to PubSub, the pool could potentially be useful.