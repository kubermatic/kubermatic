# Logging

We are using [zap](https://github.com/uber-go/zap) as logging library.
Zap is a library for structured logging and supports logging to JSON or console.

JSON should be used in production so we can fully utilize the structured logs in Kibana:

```json
{"level":"info","time":"2019-04-29T15:23:06.186+0200","caller":"test/example.go:16","msg":"Info"}
```

## Usage

### Levels

We should use:

- `log.Error` for logging errors
- `log.Info` for logging state changes which might be relevant for the operator
- `log.Debug` for infos which are targeted in aiding debugging. Those are not being shown in production.

### Enriching logs with fields

In the majority of cases, there is no need for you to create a new logger from scratch. Instead, we have have a single logger instance as an entrypoint which we pass to all required components.
Each component can then build on the passed logger by adding new fields.
This enables us to retain parent fields without the need to specify them on each log statement.

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

### Logging Conventions

The subsequent cases are general conventions, which apply to the majority of use-cases:

- For logging errors, include the passed up error as field in the log message:

  ```go
  // zap.Error is a helper method which sets the error as field "error" & prints the error based on the available implementations.
  // For more details: https://github.com/uber-go/zap/blob/master/error.go#L38
  log.Errorw("Failed to reconcile cluster", zap.Error(err))
  ```

- Avoid templating in the error message. Instead add variable information as a field:
  
  ```go
  // instead of
  log.Infof("Secret %q already deleted", secret.Name)

  // do this
  log := log.With("secret", secret.Name)
  log.Info("secret was already deleted")
  // will output:
  // {"level":"info","time":"2022-04-28T18:22:43.084+0200","caller":"application-secret-synchronizer/controller.go:144","msg":"secret was already deleted,"secret":"secret-1"}
  ```

  This plays nicely with external packages we use (e.g. reconciler), and makes it easier to filter in log aggregation systems like Grafana Loki or Kibana.

## Handling old code

We've not followed that guidelines in the past, thus we might encounter code which does not apply to the above mentioned guidelines.
This code should be converted as soon as it gets modified.
It is the responsibility of the individual PR author & reviewer to make sure, modified code parts adhere to the guidelines.
