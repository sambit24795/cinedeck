# Contributing

## Checks

Before opening a pull request, run the full check suite and make sure all three
commands pass. `gofmt -l .` must print nothing.

```
gofmt -l . && go vet ./... && go test ./...
```

## Scope

cinedeck only supports legal sources. Keep contributions to legal sources only;
do not add piracy indexers or any other source that encourages copyright
infringement.
