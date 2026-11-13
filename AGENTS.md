# Agent Instructions

## Publishing a new release / QPKG

Publishing is fully automated by `.github/workflows/release.yml`, which is
triggered by **creating a GitHub release** (not by pushing to `master`).

1. Land the changes on `master` (`git push origin master`).
2. Pick the next version using semver against the latest tag
   (`gh release list --limit 1`):
   - new user-facing feature -> minor bump (e.g. `v1.2.1` -> `v1.3.0`)
   - bug fix / dependency bump only -> patch bump (e.g. `v1.2.1` -> `v1.2.2`)
3. Create the release, which triggers the build:

   ```sh
   gh release create vX.Y.Z --target master --title vX.Y.Z --generate-notes
   ```

The workflow builds the binary for all 5 architectures (amd64, arm64,
armv5/v6/v7), builds the `.qpkg` for each via QDK, and uploads them as
release assets. The version is taken from the git tag (`github.ref_name`)
and substituted into `qpkg/qpkg.cfg` at build time, so `QPKG_VER` stays
`0.0.0` in the repo.

Verify after the run completes:

```sh
gh run list --workflow=release.yml --limit 1
gh release view vX.Y.Z --json assets --jq '.assets[].name'
```

Expect 5 `QNAPExporter_vX.Y.Z_<arch>.qpkg` assets plus the per-arch
`qnapexporter-*.tar.gz` binaries.

Once the release assets are published, `.github/workflows/qnap-repo.yml` is
triggered automatically (on the `published` event) and regenerates
`repo.xml` on GitHub Pages at:
`https://pedropombeiro.github.io/qnapexporter/repo.xml`

**One-time setup required**: GitHub Pages must be enabled with source set to
"GitHub Actions" in the repo Settings → Pages before the first deploy will
succeed. This is a manual step done once in the GitHub UI.

## Mise task metadata

When creating or updating a mise task, maintain its argument and completion metadata
alongside its implementation. Use `#USAGE` directives for file tasks or `usage` for
TOML tasks, with help text, repeatability, fixed `choices`, and dynamic `complete`
rules where applicable. Derive completions from authoritative local metadata.
Completion must not read credentials, change state, or contact live services.

For wrappers whose CLI parses forwarded arguments, set `raw_args = true` (or
`#MISE raw_args=true`) and keep forwarding intact. `raw` controls terminal access,
not argument parsing. Tasks without arguments need only task-name completion.

Validate flags with `mise __complete_word__ --shell zsh --line 'mise run TASK --'`
and test representative value prefixes. Test forwarding with harmless fixtures,
not operational tasks. Complete immediately after `--` or a value prefix; a
standalone `--` followed by a space disables task-aware completion.

## Git hooks

This repo uses [hk](https://hk.jdx.dev) (`hk.pkl`) as its git hook manager. List the
current steps with `hk check --plan`; do not mirror them here. The `linters` mapping in
`hk.pkl` is shared by every hook:

- `pre-commit`: fixes staged files and re-stages them (stashes unstaged changes).
- `commit-msg`: requires a Conventional Commits subject
  (`scripts/validate-conventional-commit.sh`).
- `pre-push`: the same checks plus `go test ./...`.
- `fix`: fixes files in place without stashing. `mise run fix [files]` and the Claude Code
  post-edit hook (`.claude/settings.json`) use it.
- `check`: check-only, used by `mise run lint [files]` and CI (`hk check --all`).

OpenCode formats edited files with its built-in `gofmt` and `shfmt` formatters, so the
repo has no OpenCode plugin. The `vendor/` and `bin/` directories are excluded via
`hk.pkl`'s `exclude` setting, and YAML rules are relaxed for GitHub Actions workflows in
`.yamllint.yml`.

Hooks are wired through the tracked `.hk-hooks/` wrappers; run `mise run install` (which
sets `core.hooksPath` to `.hk-hooks`) to enable them. `hk` and the other tools are
provided by `mise.toml`. Bypass the hooks for an individual commit with `HK=0` when
needed.
