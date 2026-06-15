KubeFlex provides an end-to-end (E2E) test suite that validates full system behavior in a Kubernetes environment.

The end-to-end test suite can be used manually by a contributor and also is used in CI.

The end-to-end test suite can either test the local sources or a given release.

## Prerequisites
`helm`,`jq` and  `kubectl` are required to run the test.
Also, either `kind` or `k3d`, depending on the sort of host cluster chosen.

## Run E2E tests manually

From the root directory of this git repository, you can run any of the following commands. The `--platform` and `--release` options combine orthogonally.

```shell
test/e2e/run.sh   # Run E2E tests against local source in kind cluster
test/e2e/run.sh --release latest   # Run E2E tests against the latest released version
test/e2e/run.sh --release v0.9.1   # Run E2E tests against a specific release
test/e2e/run.sh --platform k3d     # Run E2E tests against local source in k3d cluster
```
