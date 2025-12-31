KubeFlex provides an end-to-end (E2E) test suite that validates full system behavior in a Kubernetes environment.

End-to-end tests that can be used manually by a contributor or be triggered by CI.

## Prerequisites
`kind` and `kubectl` are required to run the test.

**The E2E tests can run against either:** 
  - local source builds 
  - a released KubeFlex version, specified with the `--release` option
   
When a release is specified, the tests install KubeFlex from the published Helm chart. The special value latest installs the most recent released version, while a literal version installs that specific release.

From the root directory of this git repository, you can run:
```shell
test/e2e/run.sh   # Run E2E tests against local source
test/e2e/run.sh --release latest   # Run E2E tests against the latest released version
test/e2e/run.sh --release v0.9.2     # Run E2E tests against a specific release
```
