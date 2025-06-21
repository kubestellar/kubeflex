# k3s 

## Code design

`pkg/k3s` package adopts a **component driven** design which implies the k3s code to be splitted into files name after k3s components:

- `apiserver.go` k3s API server
- `reconciler.go` k3s reconciler which expose `Reconcile` function that reconcile all k3s components
- `service.go` k3s service

Each Go file contains all methods related to its component.