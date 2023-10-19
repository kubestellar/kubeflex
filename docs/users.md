# User's Guide

## Installation

[kind](https://kind.sigs.k8s.io) and [kubectl](https://kubernetes.io/docs/tasks/tools/) are 
required. Note that we plan to add support for other Kube distros. A hosting kind cluster 
is created automatically by the kubeflex CLI.

Download the latest kubeflex CLI binary release for your OS/Architecture from the 
[release page](https://github.com/kubestellar/kubeflex/releases) and copy it
to `/usr/local/bin` or another location in your `$PATH`.

If you have [Homebrew](https://brew.sh), use the following commands to install kubeflex:

```shell
brew tap kubestellar/kubeflex https://github.com/kubestellar/kubeflex
brew install kubeflex
```

Once the CLI is installed, the following CLI command creates a kind cluster and installs
the KubeFlex operator:

```shell
kflex init --create-kind
```
## Install KubeFlex on an existing cluster

You can install KubeFlex on an existing cluster with nginx ingress configured for SSL passthru.
At this time, we have only tested this option with Kind. To create a kind cluster with nginx ingress,
follow the instructions [here](https://kind.sigs.k8s.io/docs/user/ingress/). Once you have 
your ingress running, you will need to configure nginx ingress for SSL passthru. Run the command:

```shell
kubectl edit deployment ingress-nginx-controller -n ingress-nginx
```

and add `--enable-ssl-passthrough` to the list of args for the container named `controller`.

## Use a different DNS service

To use a different domain for DNS resolution, you can specify the `--domain` option when 
you run `kflex init`. This domain should point to the IP address of your ingress controller, 
which handles the routing of requests to different control plane instances based on the hostname. 
A wildcard DNS service is recommended, so that any subdomain of your domain (such as *.<domain>) 
will resolve to the same IP address. The default domain in KubeFlex is localtest.me, which is a
wildcard DNS service that always resolves to 127.0.0.1. 
For example, `cp1.localtest.me` and `cp2.localtest.me` will both resolve to your local machine.

## Creating a new control plane

You can create a new control plane using the KubeFlex CLI or using any Kubernetes client or `kubectl`.

To create a new control plane with name `cp1` using the KubeFlex CLI:

```shell 
kflex create cp1
```

The KubeFlex CLI applies a `ControlPlane` CR, then waits for the control plane to become available
and finally it retrieves the `Kubeconfig` file for the new control plane, merges it with the current
Kubeconfig and sets the current context to the new control plane context.

At this point you may interact with the new control plane using `kubectl`, for example:

```shell
kubectl get ns
kubectl create ns myns
```
to switch the context back to the hosting cluster context, you may use the `ctx` command:

```shell
kflex ctx
```

To switch back to a control plane context, use the 
`ctx <control plane name>` command, e.g:

```shell
kflex ctx cp1
```

The same result can be accomplished with kubectl by using the `ControlPlane`` CR, for example:


```shell
kubectl apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: cp1
spec:
  backend: shared
  type: k8s
EOF
```

After applying the CR to the hosting kind cluster, you may check the status according to the usual
kubernetes conventions:

```shell
$ kubectl get controlplanes
NAME   SYNCED   READY   AGE
cp1    True     True    5d18h
```

In the above example, `SYNCED=True` means that all resources required to run the control plane
have been succssfully applied on the hosting cluster, and `READY=True` means that the Kube APIServer
for the control plane is available. You may also use `kubectl describe` to get more info about the
control plane.

To delete a control plane, you just have to delete the CR for that control plane, for example
using `kubectl delete controlplane cp1`. However, if you created the control plane with the `kflex`
CLI it would be better to use the `kflex` CLI so that it will remove the Kubeconfig for the control plane
from the current Kubeconfig and switch the context back to the context for the hosting cluster.

To delete a control plane with the kubeflex CLI use the command:

```shell
kubectl delete <control-plane-name>
```

If you are not using the kflex CLI to create the control plane and require access to the control plane,
you may retrieve the secret containing the control plane Kubeconfig, which is hosted in the control
plane hosting namespace (by convention `<control-plane-name>-system`) and is named `admin-kubeconfig`.

For example, the following commands retrieves the Kubeconfig for the control plane `cp1`:

```shell
NAMESPACE=cp1-system
kubectl get secrets -n ${NAMESPACE} admin-kubeconfig -o jsonpath='{.data.kubeconfig}' | base64 -d
```

### Accessing the control plane from within a kind cluster

For control plane of type k8s, the Kube API client can only use the 127.0.0.1 address. The DNS name 
`<control-plane-name>.localtest.me`` is convenient for local test and dev but always resolves to 127.0.0.1, that does not work in a container. For accessing the control plane from within the KubeFlex hosting
cluster, you may use the controller manager Kubeconfig, which is maintained in the secret with name
`cm-kubeconfig` in the namespace hosting the control plane, or you may use the Kubeconfig in the 
`admin-kubeconfig` secret with the address for the server `https://<control-plane-name>.<control-plane-namespace>:9443`.

To access the control plane API server from another kind cluster on the same docker network, you
can find the value of the nodeport for the service exposing the control plane API service, and construct
the URL for the server as `https://kubeflex-control-plane:<nodeport>`


## Control Plane Types

At this time KubFlex supports the following control plane types:

- k8s: this is the stock Kube API server with a subset of controllers running in the controller manager.
- ocm: this is the [Open Cluster Management Multicluster Control Plane](https://github.com/open-cluster-management-io/multicluster-controlplane), which provides a basic set of capabilities such as 
clusters registration and support for the [`ManifestWork` API](https://open-cluster-management.io/concepts/manifestwork/).
- vcluster: this is based on the [vcluster project](https://www.vcluster.com) and provides the ability to create pods in the hosting namespace of the hosting cluster.

## Control Plane Backends

KubeFlex roadmap aims to provide different types of backends: shared, dedicated, and for
each type the ability to choose if etcd or sql. At this time only the following 
combinations are supported based on control plane type:

- k8s: shared postgresql
- ocm: dedicated etcd
- vcluster: dedicated sqlite

## Creating with a selected control plane type

If you are using the kflex CLI, you can use the flag `--type` or `-t` to select a particular
control plane type. If this flag is not specified, the default `k8s` is used.

To create a control plane of type `vcluster` run the command:

```shell
kflex create cp2 --type vcluster
```

To create a control plane of type `ocm` run the command:

```shell
kflex create cp3 --type ocm
```

## Working with an OCM control plane

Let's create an OCM control plane:

```shell
$ kflex create cp3 --type ocm
✔ Checking for saved initial context...
✔ Switching to initial context...
✔ Creating new control plane cp3...
✔ Waiting for API server to become ready...
```

We may check the CRDs available for the OCM control plane:

```shell
$ kubectl get crds
NAME                                                           CREATED AT
addondeploymentconfigs.addon.open-cluster-management.io        2023-07-08T21:17:44Z
addonplacementscores.cluster.open-cluster-management.io        2023-07-08T21:17:44Z
clustermanagementaddons.addon.open-cluster-management.io       2023-07-08T21:17:44Z
managedclusteraddons.addon.open-cluster-management.io          2023-07-08T21:17:44Z
managedclusters.cluster.open-cluster-management.io             2023-07-08T21:17:44Z
managedclustersetbindings.cluster.open-cluster-management.io   2023-07-08T21:17:44Z
managedclustersets.cluster.open-cluster-management.io          2023-07-08T21:17:44Z
manifestworks.work.open-cluster-management.io                  2023-07-08T21:17:44Z
placementdecisions.cluster.open-cluster-management.io          2023-07-08T21:17:44Z
placements.cluster.open-cluster-management.io                  2023-07-08T21:17:44Z
```

We may also register clusters with the OCM control plane and deploy workloads
using the `ManifestWork` API. In order to do that, you need first to install
the Open Cluster Management [clusteradm CLI](https://open-cluster-management.io/getting-started/installation/start-the-control-plane/), e.g.

```shell
curl -L https://raw.githubusercontent.com/open-cluster-management-io/clusteradm/main/install.sh | bash
```

With the current context set to the ocm control plane, we can use `clusteradm` to retrieve a token
used to register managed clusters:

```shell
$ clusteradm get token --use-bootstrap-token
clusteradm join --hub-token <some value> --hub-apiserver https://cp3.localtest.me:9443/ --cluster-name <cluster_name> 
```

The command returns the command to run on the managed cluster (actual token value not shown in example).

Now create a kind cluster to register with ocm, with the command:

```shell
kind create cluster --name cluster1
```

Once the cluster is ready, run the command above, taking care of replacing <cluster_name> with cluster1
and leaving the actual token value. Most importantly, make sure to add the flag `--force-internal-endpoint-lookup` which allows the managed cluster to communicate with the OCM control plane
using the docker network that all kind clusters share. Note that the `kind create cluster` command
switches the context to the new cluster `cluster`, so the `clusteramd join` command is run using the
new cluster context.

```shell
clusteradm join --hub-token <some value> --hub-apiserver https://cp2.localtest.me:9443/ --cluster-name cluster1 --force-internal-endpoint-lookup
```

At this point, switch back the context to the OCM control plane with the command:

```shell
kflex ctx cp3
```

and verifies that a Certificate Signing Request (csr) has been created on the OCM control plane
running the command `kubectl get csr`. The CSR request is part of the mechanism used by OCM
to register a new cluster. You should see an output simlar to the following:

```shell
$ kubectl get csr
NAME             AGE   SIGNERNAME                            REQUESTOR                 REQUESTEDDURATION   CONDITION
cluster1-zx5x5   7s    kubernetes.io/kube-apiserver-client   system:bootstrap:j5bork   <none>              Pending
```

Approve the csr to complete the cluster registration with the command:

```shell
clusteradm accept --clusters cluster1
```

You can now see the new cluster in the OCM inventory:

```shell
$ kubectl get managedclusters
NAME       HUB ACCEPTED   MANAGED CLUSTER URLS                  JOINED   AVAILABLE   AGE
cluster1   true           https://cluster1-control-plane:6443   True     True        3m25s
```

Finally, you may deploy a workload on the managed cluster using the ManifestWork API:

```shell
kubectl apply -f - <<EOF
apiVersion: work.open-cluster-management.io/v1
kind: ManifestWork
metadata:
  namespace: cluster1
  name: my-first-work
spec:
  workload:
    manifests:
      - apiVersion: v1
        kind: ServiceAccount
        metadata:
          namespace: default
          name: my-sa
      - apiVersion: apps/v1
        kind: Deployment
        metadata:
          namespace: default
          name: nginx-deployment
          labels:
            app: nginx
        spec:
          replicas: 3
          selector:
            matchLabels:
              app: nginx
          template:
            metadata:
              labels:
                app: nginx
            spec:
              serviceAccountName: my-sa
              containers:
                - name: nginx
                  image: nginx:1.14.2
                  ports:
                    - containerPort: 80
EOF
```
To check the workload has been deployed, switch context back to the managed cluster
and list deployments:

```shell
kflex ctx kind-cluster1
```

```shell
$ kubectl get deployments.apps 
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
nginx-deployment   3/3     3            3           20s
```

## Working with an vcluster control plane

Let's create a vcluster control plane:

```shell
$ kflex create cp2 --type vcluster
✔ Checking for saved initial context...
✔ Creating new control plane cp2...
✔ Waiting for API server to become ready...
```

Now interact with the new control plane, for example creating a new nginx pod:

```shell
kubectl run nginx --image=nginx
```

Verify the pod is running:

```shell
$ kubectl get pods
NAME    READY   STATUS    RESTARTS   AGE
nginx   1/1     Running   0          24s
```

Access the pod logs:

```shell
$ kubectl logs nginx 
/docker-entrypoint.sh: /docker-entrypoint.d/ is not empty, will attempt to perform configuration
/docker-entrypoint.sh: Looking for shell scripts in /docker-entrypoint.d/
...
```

Exec into the pod and run the `ls` command:

```shell
$ kubectl exec -it nginx -- sh
# ls
bin   dev                  docker-entrypoint.sh  home  media  opt   product_uuid  run   srv  tmp  var
boot  docker-entrypoint.d  etc                   lib   mnt    proc  root          sbin  sys  usr
```

Switch context back to the hosting Kubernetes and check that pod is running in the `cp2-system`
namespace:

```shell
kflex ctx
```

```shell
$ kubectl get pods -n cp2-system 
NAME                                                READY   STATUS    RESTARTS   AGE
coredns-64c4b4d78f-2w9bx-x-kube-system-x-vcluster   1/1     Running   0          6m58s
nginx-x-default-x-vcluster                          1/1     Running   0          4m26s
vcluster-0                                          2/2     Running   0          7m15s
```

The nginx pod is the one with the name `nginx-x-default-x-vcluster`.


