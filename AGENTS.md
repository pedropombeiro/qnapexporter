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

   ```
   gh release create vX.Y.Z --target master --title vX.Y.Z --generate-notes
   ```

The workflow builds the binary for all 5 architectures (amd64, arm64,
armv5/v6/v7), builds the `.qpkg` for each via QDK, and uploads them as
release assets. The version is taken from the git tag (`github.ref_name`)
and substituted into `qpkg/qpkg.cfg` at build time, so `QPKG_VER` stays
`0.0.0` in the repo.

Verify after the run completes:

```
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

## Git hooks

This repo uses [pre-commit](https://pre-commit.com) (`.pre-commit-config.yaml`:
go-fmt, go-vet, go-mod-tidy, golangci-lint). A global hk hook layer may also
run from the user's dotfiles; it is not part of this repo and can be bypassed
with `HK=0` for individual commits when it produces false positives (e.g.
editorconfig-checker flagging Go tabs, or shfmt reformatting shell scripts).
