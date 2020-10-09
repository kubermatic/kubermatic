# Logging

We are using [zap](https://github.com/uber-go/zap) as logging library.
Zap is a library for structured logging and supports logging to JSON or console.

JSON should be used in production so we can fully utilize the structured logs in Kibana:
```json
{"level":"info","time":"2019-04-29T15:23:06.186+0200","caller":"test/example.go:16","msg":"Info"}
```
Console is for local development:
```
2019-04-29T15:24:02.285+0200	info	test/example.go:16	Info
```

## Guidelines

There are 3 things you will log:
- Errors
- Infos
- Debug

## Usage

### Levels

We should use:
- `log.Error` for logging errors
- `log.Info` for logging state changes which might be relevant for the operator
- `log.Debug` for infos which are targeted in aiding debugging. Those are not being shown in production.

### Enriching logs with fields
To fully utilize the logger, we will have a single logger instance as entrypoint which we pass to all required components.
Each component can then create a new logger which inherits the previous logger.
This enables us to add fields to the logger - without specifying them on each log statement:
```go
//log got passed in to the component
clusterLog := log.With("cluster", "some-cluster")
clusterLog.Info("Something good happened")
// Will output:
// {"level":"info","time":"2019-04-29T15:23:06.186+0200","caller":"test/example.go:24","msg":"Something good happened","cluster":"some-cluster"}
```

To add fields for a single statement:
```go
// Derive a new log object
log.With("cluster", "some-cluster").Info("Something good happened")
// Use the "*w" functions of the SugardLogger
log.Infow("Something good happened", "cluster", "some-cluster")
// Will output:
// {"level":"info","time":"2019-04-29T15:23:06.186+0200","caller":"test/example.go:24","msg":"Something good happened","cluster":"some-cluster"}
// {"level":"info","time":"2019-04-29T15:23:06.186+0200","caller":"test/example.go:24","msg":"Something good happened","cluster":"some-cluster"}
```

For logging errors include the passed up error as field in the log message:
```go
// zap.Error is a helper method which sets the error as field "error" & prints the error based on the available implementations.
// For more details: https://github.com/uber-go/zap/blob/master/error.go#L38
log.Errorw("Failed to reconcile cluster", zap.Error(err))
```

## Handling old code

We've not followed that guidelines in the past, thus we might encounter code which does not apply to the above mentioned guidelines.
This code should be converted as soon as it gets modified.
It is the responsibility of the individual PR author & reviewer to make sure, modified code parts adhere to the guidelines.
