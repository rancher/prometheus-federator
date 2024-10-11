# Build Process For Project Operators

As a type of [Project Operator](https://github.com/rancher/helm-project-operator), Prometheus Federator is primarily composed of three components:
- The Underlying Helm Chart (Build Dependency)
- The Project Operator Image (Build Dependency)
- The Project Operator Helm Chart (What A User Actually Deploys)

## Underlying Helm Chart

The underlying Helm Chart whose releases are deployed on behalf of every registered ProjectHelmChart CR is found in [`rancher-project-monitoring`](../packages/rancher-project-monitoring/).

While the source of this chart is found in the `packages/` directory, which is a construct of any [rancher/charts-build-scripts](https://github.com/rancher/charts-build-scripts) repository (see [the docs on Packages](https://github.com/rancher/charts-build-scripts/blob/master/templates/template/docs/packages.md) for more information), it is expected that a developer who files a PR with changes will run the `make charts` command to ensure that the package is read by the `rancher/charts-build-scripts` binary to **produce / auto-generate** the Helm Charts and manage the `assets/`+`charts/` directories as well as the `index.yaml` entries to introduce this package in a standard Helm repository fashion.

Once `make charts` has been run and the chart is built from `packages/rancher-project-monitoring/` -> `charts/rancher-project-monitoring/${VERSION}` (part of the `make charts` command), the built chart is then converted into a `.tgz.base64` version of itself in [scripts/build-chart](../../scripts/build-chart) and left in `bin/rancher-project-monitoring/rancher-project-monitoring.tgz.base64`.

```bash
helm package charts/${CHART}/${VERSION} --destination bin/${CHART}
base64 bin/${CHART}/${CHART}-${VERSION}.tgz > bin/${CHART}/${CHART}.tgz.base64
rm bin/${CHART}/${CHART}-${VERSION}.tgz
```

## The Project Operator Image

To implement a Project Operator, Helm Project Operator expects a user to run the `operator.Init` command, which appears in Prometheus Federator's [`main.go`](../../main.go) as follows:

```go
operator.Init(ctx, f.Namespace, cfg, common.Options{
    OperatorOptions: common.OperatorOptions{
        HelmAPIVersion:   HelmAPIVersion,
        ReleaseName:      ReleaseName,
        SystemNamespaces: SystemNamespaces,
        ChartContent:     base64TgzChart,
        Singleton:        true, // indicates only one HelmChart can be registered per project defined
    },
    RuntimeOptions: f.RuntimeOptions,
})
```

While the `HelmAPIVersion`, `ReleaseName`, and `SystemNamespaces` supplied are hard-coded into the [`main.go`](../../main.go) and the `RuntimeOptions` are taken from the CLI arguments provided, the only additional value that is needed to build this chart is the `.tgz.base64` version of the chart that is passed in as a string to the operator.

This is precisely what we build in the prior step at `bin/rancher-project-monitoring/rancher-project-monitoring.tgz.base64`, which is why that path is found as a `go embed` directive on building the `main.go`:

```go
//go:embed bin/rancher-project-monitoring/rancher-project-monitoring.tgz.base64
base64TgzChart string
```

Once your [`main.go`](../../main.go) is ready to be built, you can run `./scripts/build`, which will run the underlying `go build` command and place the created binary in `bin/prometheus-federator`.

Once the binary has been created, it is then packaged into a container image in the [`scripts/package`](../../scripts/package) step, where we build the Dockerfile found in `packages/Dockerfile` to produce the final image.

## The Project Operator Helm Chart

This is the component that the average user is actually expected to directly deploy; it is also maintained in the `packages/` directory, like the Underlying Helm Chart.

As explained above, packages are a construct of any [rancher/charts-build-scripts](https://github.com/rancher/charts-build-scripts) repository (see [the docs on Packages](https://github.com/rancher/charts-build-scripts/blob/master/templates/template/docs/packages.md) for more information), so just like with the Underlying Helm Chart, it is expected that a developer who files a PR with changes will run the `make charts` command to ensure that the package is read by the `rancher/charts-build-scripts` binary to **produce / auto-generate** the Helm Charts and manage the `assets/`+`charts/` directories as well as the `index.yaml` entries to introduce this package in a standard Helm repository fashion.

Once `make charts` has been run and the chart is built from `packages/prometheus-federator` -> `charts/prometheus-federator/${VERSION}` (part of the `make charts` command), the chart is now visible on the Helm repository maintained within your fork!

## TLDR; Putting It All Together

Therefore, as a whole, the build process of Underlying Helm Chart looks as follows:
- By a developer on making a PR to change the Underlying Helm Chart:
    - Run `make charts` to produce the Underlying Helm Chart in `charts/rancher-project-monitoring/${VERSION}` from `packages/rancher-project-monitoring/`
- By running `make` (which runs [rancher/dapper](https://github.com/rancher/dapper) on the `Dockerfile.dapper`, which in turn runs [`./scripts/ci`](../../scripts/ci) that runs the following commands in an container image):
    - Run `./scripts/build-chart` to produce `bin/rancher-project-monitoring/rancher-project-monitoring.tgz.base64` from the Underlying Helm Chart
    - Run `./scripts/build` to produce the Project Operator Binary `bin/prometheus-federator`; **this will work since `bin/rancher-project-monitoring/rancher-project-monitoring.tgz.base64` exists from the previous step**
    - Run `./scripts/package` to produce the Project Operator Image
- By a developer on making a PR to change the Project Operator Helm Chart:
    - Make changes in `packages/rancher-project-monitoring/` (such as updating the `values.yaml` or `Chart.yaml` to point to the latest Project Operator Image on a change that has been made)
    - Run `make charts` to produce `charts/rancher-project-monitoring/${VERSION}` from `packages/rancher-project-monitoring/`
    - Run `./scripts/build-chart` to produce `bin/rancher-project-monitoring/rancher-project-monitoring.tgz.base64` from `charts/rancher-project-monitoring/${VERSION}`
