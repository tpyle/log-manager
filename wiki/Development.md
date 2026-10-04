# Development and Releasing

## Module layout

The repository contains four Go modules:

| Directory | Module path | Tag prefix |
|-----------|-------------|------------|
| `/` | `github.com/tpyle/log-manager/v3` | `v3.x.y` |
| `/zerologmgr` | `github.com/tpyle/log-manager/zerologmgr/v3` | `zerologmgr/v3.x.y` |
| `/logrusmgr` | `github.com/tpyle/log-manager/logrusmgr/v3` | `logrusmgr/v3.x.y` |
| `/examples` | `github.com/tpyle/log-manager/examples` | not released |

The adapters are separate modules so the root `go.mod` has no requirements.
Depending on the core never adds zerolog or logrus to a program's module
graph, `go.sum`, or dependency scans. `TestRootModuleHasNoRequirements` fails
if a requirement is added to the root module.

The `/v3` suffix comes after the adapter directory
(`.../zerologmgr/v3`, not `.../v3/zerologmgr`). Go maps a nested module's
path to a repository directory and only recognizes a major-version suffix as
the last path element, so `.../v3/zerologmgr` would have to be in a `v3/zerologmgr` directory.

## Local development

Each adapter's `go.mod` requires a released core version and has a
`replace` directive pointing at `../`. The `examples` module has
`replace` directives for all three. This way, changes to the core are tested
against the adapters immediately. Go ignores `replace` directives in
dependencies, so consumers use the required version.

Run commands per module:

```bash
for m in . zerologmgr logrusmgr examples; do
  (cd $m && go mod tidy -diff && go vet ./... && go test -race -cover ./...)
done
```

CI runs the same steps for each module.

## Releasing

Tag the core first, because the adapters require a released core version:

```bash
git tag v3.0.0
git tag zerologmgr/v3.0.0
git tag logrusmgr/v3.0.0
git push origin v3.0.0 zerologmgr/v3.0.0 logrusmgr/v3.0.0
```

The adapters' versions don't have to match the core, but keeping the major
version in step makes compatibility clear.

If an adapter starts using a new core API, raise its `require` line to the
core version that introduced the API, and release that core version before
tagging the adapter. Because the `replace` directive hides a missing core
release, check that the adapter builds against the released core before
tagging it:

```bash
cd zerologmgr
go mod edit -dropreplace github.com/tpyle/log-manager/v3
go build ./... && go test ./...
git checkout go.mod go.sum
```
