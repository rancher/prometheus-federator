# Integration Test

To get started you need a kubeconfig and ginkgo installed. To install the matching ginkgo version from go.mod run from the repo root : `go install github.com/onsi/ginkgo/v2/ginkgo`

The easiest way to get familiar with the integration tests is to use `k3d`:

```sh
k3d cluster create test-cluster
k3d kubeconfig get test-cluster > kubeconfig.yaml
export KUBECONFIG=$(pwd)/kubeconfig.yaml
```

To run the actual tests:

```sh
COVER=true KUBECONFIG=$KUBECONFIG ./scripts/integration
```

You can run isolated test cases using

```sh
KUBECONFIG=$KUBECONFIG FOCUS="HPO/Namespace" ./scripts/integration
```

By default, for idempotency, the tests cleanup resources they create to be able to re-run tests without cleanup. To Disable cleanup you can pass the environment variable: `DISABLE_CLEANUP=true`, which will persist resources created by the tests for debugging potential problems/bugs.

The entry point of the integration tests are in the `./internal/integration` package. 

Matching integration suites are usually defined in particular sub-packages with the filename `integration.go`. This was originally done so we can re-use private methods of that package, and improve the DEV UX to developping inside the appropriate package when designing tests.

:warning: using `go test` is possible from the `internal/integration` package, but will not run suites in parallel, so instead of taking 30-60 seconds, it could take upward of 5mins to run all the tests depending on your local hardware.

## IDE

The integration tests are hidden by a go build tags `integration`. For your IDE to have LSP support for these packages requires adding the flag to your default build tags.

For example, for VS Code you need to add to settings.json:
```json
  "go.buildTags": "integration",
```

## Debugging

Unlike testing frameworks like `testify` which directly extend go's built-in `testing` package which allows you debug individual tests from your IDE, you need to replace in `integration_suite_test.go`:

```go
	RunSpecs(t, "Integration Suite")
```

with:

```go
  suiteConfig, reporterConfig := GinkgoConfiguration()
	// // suiteConfig.LabelFilter = "DEBUG"
  suiteConfig.FocusStrings = []string{"HPO/Namespace/Init"}
  RunSpecs(t, "Integration Suite", suiteConfig, reporterConfig)
```

to focus debugging particular test suites. It is recommended to do so since the debug process launches go test, which  cannot parallelize the test suites we provide.