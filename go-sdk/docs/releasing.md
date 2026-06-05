# Releasing

## Preconditions

- All tests pass: `go test ./...`
- Module metadata is correct in `go.mod`
- Changelog or release notes are updated

## Release With the Monorepo Action

Update the Go SDK release manifest:

```yaml
# go-sdk/release.yml
go_sdk:
  version: v0.1.0
```

Merge that change to `main`. The root `Release Go SDK` workflow validates the
version, runs `go test ./...`, checks that the tag does not already exist, and
pushes a prefixed module tag:

```text
go-sdk/v0.1.0
```

The Go module path is `github.com/pvium/sdks/go-sdk`, so monorepo tags must use
the `go-sdk/` prefix. Do not tag Go SDK releases as plain root tags like
`v0.1.0`.

## Manual Fallback

```bash
go test ./...
git tag go-sdk/v0.1.0
git push origin go-sdk/v0.1.0
```

## Recommended initial strategy

- `v0.1.x` for rapid iteration while API settles
- `v1.0.0` only when the exported API is considered stable
