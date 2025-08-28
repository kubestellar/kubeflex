# k3s 

## Code design

`pkg/k3s` package adopts a **component driven** design which implies the k3s code to be splitted into files name after k3s components:

- `apiserver.go` k3s API server
- `reconciler.go` k3s reconciler which expose `Reconcile` function that reconcile all k3s components
- `service.go` k3s service

Each Go file contains all methods related to its component.

## Install k3s as a controlplane

It is decided to use [k3s docker image](https://docs.k3s.io/advanced#running-k3s-in-docker) to ease the process avoiding to use Helm (as done on `vcluster`).

Therefore, the official kubernetes client-go is enough to deploy k3s in a [`StatefulSet`](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/). This implies:
- Create a [headless service](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services)