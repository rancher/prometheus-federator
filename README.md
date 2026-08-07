# automation-core

This branch holds the CI/release plumbing shared across `prometheus-federator`'s long-lived branches (`main`, `release/v4.x`, `release/v5.x`, `release/v6.x`, ...). It doesn't carry the app source — just `.github/`, `scripts/`, `workflow-templates/`, and the couple of root files CI needs (`Makefile`, `.golangci.yaml`).

```
automation-core/
├── .github/
│   ├── actions/                     # Composite actions
│   │   ├── install-yq/
│   │   ├── build-and-push-image/
│   │   └── run-goreleaser/
│   ├── scripts/                     # Used by deploy-workflows.yml
│   │   ├── generate-commit-msg.sh
│   │   └── generate-pr-body.sh
│   └── workflows/                   # Reusable workflows, called via workflow_call
│       ├── ci.yaml                  
│       ├── lint.yaml
│       ├── integration.yaml
│       ├── prom-fed-e2e-ci.yaml
│       ├── pr-debug-publish.yaml    
│       └── deploy-workflows.yml     # generates and PRs the files below into a branch
├── workflow-templates/              # Source of truth for every workflow file a branch actually runs
│   ├── ci.yaml.tmpl
│   ├── lint.yaml.tmpl
│   ├── integration.yaml.tmpl
│   ├── prom-fed-e2e-ci.yaml.tmpl
│   ├── pr-debug-publish.yaml.tmpl
│   └── release.yaml.tmpl
├── scripts/                         # App build/package/test scripts, see note below
└── Makefile
```

## How this actually gets used

Every workflow file that lives in a release branch's `.github/workflows/` — `ci.yaml`, `lint.yaml`, `integration.yaml`, `prom-fed-e2e-ci.yaml`, `pr-debug-publish.yaml`, `release.yaml` — is generated from a template here and rolled out by `deploy-workflows.yml`. None of them are meant to be hand-edited on `main` or a `release/v*` branch. If you need to change one, change the `.tmpl` here and redeploy.

Most of those generated files are tiny, because all they really do is call a reusable workflow that lives here (`uses: rancher/prometheus-federator/.github/workflows/ci.yaml@automation-core`). `release.yaml` and `pr-debug-publish.yaml` are the two that can't work that way, because they build and push images/artifacts and need to attest provenance on them. GitHub's attest actions record which job ran them, and if that job is a `workflow_call` job delegated to a reusable workflow, the attestation gets attributed to `automation-core` instead of the actual release.

## Workflows

- `ci.yaml` — lint + build. Takes `helm-version`; `yq-checksum-amd64`/`arm64` only matter on branches whose Dockerfiles need them (`release/v6.x` right now).
- `lint.yaml` — golangci-lint. Takes `helm-version`.
- `integration.yaml` — integration tests. Takes `helm-version` and `k3s-versions` (a JSON array, passed as a string).
- `prom-fed-e2e-ci.yaml` — the full e2e suite against a k3d cluster. Takes `helm-version`, optionally `debug`/`k3s_version`.
- `deploy-workflows.yml` — renders the templates and opens a PR against a branch, see below.

## Actions

- `install-yq` — installs yq if it's missing, checksum-verified against a version pinned in the action. No inputs.
- `build-and-push-image` — buildx setup, registry login, tag/label extraction, build and push. Outputs `digest` and `tags` so whoever calls it can attest right afterwards. Used from `release.yaml.tmpl` and `pr-debug-publish.yaml.tmpl`.
- `run-goreleaser` — checks out the repo, sets up go/yq/kubectl/helm, preps the helm charts (`make package-helm` + `make build-chart`), runs `goreleaser release --clean`.

Whichever workflow calls `build-and-push-image` or `run-goreleaser` has to put the attest step right after it, in the same job — not inside the action.

## Deploying workflows

Actions → Deploy Workflows → Run workflow, with:

- `targetBranch` — e.g. `main`, `release/v6.x`
- `helmVersion` — `v4.2.0` on main, `v3.19.5` on v4.x/v5.x/v6.x
- `k3sVersions` — the JSON array as a string
- `yqChecksumAmd64`/`yqChecksumArm64` — only needed on v6.x, leave blank otherwise
- `automationCoreRef` — defaults to `automation-core`; pin it to a tag if a branch shouldn't pick up changes automatically

It renders all six templates, checks they're valid YAML, pushes a branch, and opens a PR against `targetBranch`.

## scripts/ and check-semver

These stay per-branch instead of living only here, because they're tied to things that genuinely differ per branch — Dockerfiles, the Makefile. `release.yaml.tmpl` calls `./.github/scripts/check-semver` as a relative path on purpose, so it always runs whichever branch's own copy is checked out, not this branch's.

## Making changes

- Action or reusable-workflow logic: edit it here. Anything still pinned to `@automation-core` picks it up next run, no redeploy needed.
- Template changes: edit the `.tmpl`, then run `deploy-workflows.yml` for each branch that needs it.
