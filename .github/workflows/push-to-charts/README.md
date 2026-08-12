# push-to-charts

Automates opening PRs against [rancher/charts](https://github.com/rancher/charts) when a new prometheus-federator release is published.

## Trigger

The workflow fires on `release: published` (after goreleaser uploads the chart artifact) and can also be dispatched manually with an explicit tag.

Because `release: published` always runs from the default branch, **no backporting is needed** — a single copy of these scripts handles all version lines via `CHARTS_BRANCH_MAP` in `common.sh`.

## Adding a new prometheus-federator/Rancher version pair

Edit `common.sh` and add an entry to `CHARTS_BRANCH_MAP`:

```bash
["7"]="dev-v2.15"
```

## Local usage

```bash
./.github/workflows/push-to-charts/run-local.sh \
  --charts-dir /path/to/rancher/charts \
  --tag v7.0.1-rc.2 \
  [--dry-run] \
  [--remote upstream]
```

`--dry-run` runs all local git work (commits to your charts clone) but skips push and PR creation.

## Step sequence

| Script | What it does |
|---|---|
| `remove-previous-rc.sh` | `make remove` for the previous RC chart version. No-op if base prometheus-federator version changed or old version was not a prerelease. |
| `update-package-yaml.sh` | Updates `url` to the new release artifact. Bumps charts version (minor or patch) unless the base version is unchanged (RC→RC or RC→stable). |
| `update-patches.sh` | Updates `appVersion` in `Chart.yaml.patch`, then runs `make prepare`, `make patch`, `make clean`. |
| `generate-assets.sh` | Runs `make charts USE_CACHE=true`. |
| `update-release-yaml.sh` | Prepends the new combined version (`{charts_version}+up{pf_version}`) to the `prometheus-federator` entry in `release.yaml`. |
| `create-pr.sh` | Pushes the branch and opens a PR against the target branch in rancher/charts. |

## Version bump rules

| Transition | Charts version |
|---|---|
| `7.0.0` → `7.0.1-rc.1` | patch bump: `110.0.0` → `110.0.1` |
| `7.0.1-rc.1` → `7.0.1-rc.2` | no bump: stays `110.0.1` |
| `7.0.1-rc.2` → `7.0.1` | no bump: stays `110.0.1` |
| `7.0.1` → `7.1.0-rc.1` | minor bump: `110.0.1` → `110.1.0` |
| `7.x.x` → `8.0.0-rc.1` | major bump: `110.x.x` → `111.0.0` |

## Key env vars

| Var | Description |
|---|---|
| `TAG` | prometheus-federator release tag (e.g. `v7.0.1-rc.2`) |
| `CHARTS_DIR` | Path to local rancher/charts clone |
| `CHARTS_REMOTE` | Remote name in `CHARTS_DIR` (default: `origin`) |
| `DRY_RUN` | Set to `true` to skip push and PR creation |
| `SOURCE_REPO` | Source repo for PR body (default: `rancher/prometheus-federator`) |

## GHA prerequisites

The workflow reads a GitHub App credential from Vault at:

```
secret/data/github/repo/rancher/prometheus-federator/github/app-credentials
```

The app must have write access to `rancher/charts` to push branches and open PRs.
