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
  (cd $m && go mod tidy -diff && go vet ./... && go test -race -cover ./... && golangci-lint run ./...)
done
```

CI runs the same steps, plus golangci-lint, for each module, and reports the
combined result as the `Test and Lint` check that `trunk` requires. When adding
a module, add it to the CI matrix and to `.github/dependabot.yml`.

## Releasing

The core and both adapters are always released together with the same
version number. Go versions each module by its own tags, so a release is three
tags on the same commit:

1. Set the new version in the `require` lines for this repository's modules in
   `zerologmgr/go.mod`, `logrusmgr/go.mod` and `examples/go.mod`, and merge.
   Dependabot ignores these modules, so it never changes these lines.
2. Tag the merge commit and push the tags:
   ```bash
   V=vX.Y.Z
   git tag -a $V -m $V && git tag -a zerologmgr/$V -m $V && git tag -a logrusmgr/$V -m $V
   git push origin $V zerologmgr/$V logrusmgr/$V
   ```
3. Check from a scratch module that
   `go get github.com/tpyle/log-manager/zerologmgr/v3@$V` resolves.

Because all three tags are on one commit, an adapter's required core version
always exists once the release is pushed.

## Wiki

`wiki/` is published to the GitHub wiki by the Wiki workflow whenever it
changes on `trunk`. Links between pages are written as `Page.md` so they work
when browsing the repository, and the workflow rewrites them to wiki links.
Edits made in the GitHub wiki editor are overwritten on the next sync.

The workflow can't create the wiki. Before it runs for the first time, create
any page in the repository's Wiki tab.
