End-to-end tests that can be used manually by a contributor or be triggered by CI.

## Prerequisites
`kind` and `kubectl` are required to run the test.
## End-to-End (E2E) Testing

KubeFlex provides an end-to-end (E2E) test suite that validates full system behavior in a Kubernetes environment.

**The E2E tests can run against either:** 
  - local source builds 
  - Released KubeFlex artifacts using the --release option
   

When a release is specified, the tests install KubeFlex from the published Helm chart. The special value latest installs the most recent released version, while a literal version installs that specific release.

In the root directory of this git repo: 
```shell
test/e2e/run.sh   # Run E2E tests against local source
test/e2e/run.sh --release latest   # Run E2E tests against the latest released version
test/e2e/run.sh --release 0.9.2     # Run E2E tests against a specific release
```
