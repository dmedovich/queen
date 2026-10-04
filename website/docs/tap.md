---
id: tap
title: Migration Tap
---

# Migration Tap

Tap streams migration events while Queen runs. It is useful for debugging, CLI live views, tests, and migration reports.

## JSON sink

```go
sink := tap.NewJSONSink(os.Stdout)
q := queen.New(postgres.New(db), queen.WithTap(sink))
```

Events include `start`, `exec`, transaction lifecycle events, and `end`.

## Capture Go-function SQL

SQL migrations emit one `exec` event for the migration SQL. For Go-function migrations, wrap the transaction:

```go
UpFunc: func(ctx context.Context, tx *sql.Tx) error {
    t := tap.ObserveTx(ctx, tx)
    _, err := t.ExecContext(ctx, `UPDATE users SET email = LOWER(email)`)
    return err
}
```

## Analyzer

```go
sink := tap.NewAnalyzerSink(
    tap.NewJSONSink(os.Stdout),
    tap.DefaultAnalyzerConfig(),
)
```

The analyzer adds operation names, normalized SQL templates, bound SQL, slow-query markers, and N+1 detection.

Built-in sinks include `NopSink`, `FuncSink`, `MultiSink`, `ChannelSink` (non-blocking and drops on overflow), `JSONSink`, and `RecorderSink` for in-memory tests. Custom destinations can implement `tap.Sink`. `tap.ObserveTx` wraps the transaction Queen already passes to Go functions; it does not create a new transaction. The older `tap.Tx` helper remains as a deprecated compatibility alias.

Analyzed `exec` events include the first SQL operation, a normalized template, bound SQL for inspection, slow and N+1 markers, and the statement index within a migration. Programmatic helpers can filter, summarize, and export recorded events:

```go
events := recorder.Events()
filter, _ := tap.ParseFilter("op:select d>100ms slow")
for _, event := range events {
    if filter.Match(event) {
        fmt.Println(event.BoundSQL)
    }
}
summary := tap.Summarize(events)
top := tap.TopQueries(events, "total", 10)
_ = summary
_ = top
_ = tap.WriteMarkdown(os.Stdout, events)
```

## Live CLI

```bash
go run ./cmd/migrate up --tap --driver postgres --dsn "$DATABASE_URL"
```

Tune thresholds:

```bash
go run ./cmd/migrate up --tap \
  --tap-slow-threshold 250ms \
  --tap-nplus1-threshold 10
```
