# Logging

We are using [zap](https://github.com/uber-go/zap) via a [wrapper](https://github.com/go-logr/zapr) at the moment as logging library.
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

There are 2 things you will log:
- Errors
- Infos 

Info logs are leveled, though only 2 levels exist. The higher the level, the higher the verbosity (More less important logs):
- 0
- 1

## Usage

### Levels

We should use:
- `log.Error` for logging errors
- `log.Info` for logging state changes which might be relevant for the operator
- `log.V(1).Info` for infos which are targeted in aiding debugging. Those are not being shown in production.

### Enriching logs with fields
To fully utilize the logger, we will have a single logger instance as entrypoint which we pass to all required components.
Each component can then create a new logger which inherits the previous logger.
This enables us to add fields to the logger - without specifying them on each log statement:
```go
//log got passed in to the component
clusterLog := log.WithValues("cluster", "some-cluster")
clusterLog.Info("Something good happened")
// Will output:
// {"level":"info","time":"2019-04-29T15:23:06.186+0200","caller":"test/example.go:24","msg":"Something good happened","cluster":"some-cluster"}
```

To add fields for a single statement:
```go
log.WithValues("cluster", "some-cluster").Info("Something good happened")
// Will output:
// {"level":"info","time":"2019-04-29T15:23:06.186+0200","caller":"test/example.go:24","msg":"Something good happened","cluster":"some-cluster"}
```

## Handling old code

We've not followed that guidelines in the past, thus we might encounter code which does not apply to the above mentioned guidelines.
This code should be converted as soon as it get's modified.
It is the responsibility of the individual PR author & reviewer to make sure, modified code parts adhere to the guidelines.
