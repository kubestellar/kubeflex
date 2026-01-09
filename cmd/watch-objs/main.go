package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	api "github.com/kubestellar/kubeflex/api/v1alpha1"
	kfclient "github.com/kubestellar/kubeflex/pkg/generated/clientset/versioned"
	kfinformers "github.com/kubestellar/kubeflex/pkg/generated/informers/externalversions"
)

const ControllerName = "ensure-label"

func main() {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}

	klog.InitFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.CommandLine.StringVar(&loadingRules.ExplicitPath, "kubeconfig", loadingRules.ExplicitPath, "Path to the kubeconfig file to use")
	pflag.CommandLine.StringVar(&overrides.CurrentContext, "context", overrides.CurrentContext, "The name of the kubeconfig context to use")
	pflag.CommandLine.StringVar(&overrides.Context.AuthInfo, "user", overrides.Context.AuthInfo, "The name of the kubeconfig user to use")
	pflag.CommandLine.StringVar(&overrides.Context.Cluster, "cluster", overrides.Context.Cluster, "The name of the kubeconfig cluster to use")
	pflag.CommandLine.StringVarP(&overrides.Context.Namespace, "namespace", "n", overrides.Context.Namespace, "The name of the Kubernetes Namespace to work in (NOT optional)")
	pflag.Parse()
	ctx := context.Background()
	logger := klog.FromContext(ctx)

	logger.V(1).Info("Start", "time", time.Now())

	pflag.CommandLine.VisitAll(func(f *pflag.Flag) {
		logger.V(1).Info("Flag", "name", f.Name, "value", f.Value.String())
	})

	if len(overrides.Context.Namespace) == 0 {
		fmt.Fprintln(os.Stderr, "Namespace must not be the empty string")
		os.Exit(1)
	} else {
		logger.Info("Focusing on one namespace", "name", overrides.Context.Namespace)
	}

	restConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	if err != nil {
		logger.Error(err, "Failed to construct resstConfig")
		os.Exit(10)
	}
	if len(restConfig.UserAgent) == 0 {
		restConfig.UserAgent = ControllerName
	} else {
		restConfig.UserAgent += "/" + ControllerName
	}

	kfClient, err := kfclient.NewForConfig(restConfig)
	if err != nil {
		logger.Error(err, "Failed to construct client")
		os.Exit(10)
	}
	sif := kfinformers.NewSharedInformerFactoryWithOptions(kfClient, 0, kfinformers.WithNamespace(overrides.Context.Namespace))
	cpPreInf := sif.Tenancy().V1alpha1().ControlPlanes()
	pchPreInf := sif.Tenancy().V1alpha1().PostCreateHooks()
	cpInformer, cpLister := cpPreInf.Informer(), cpPreInf.Lister()
	pchInformer, pchLister := pchPreInf.Informer(), pchPreInf.Lister()
	cpInformer.AddEventHandler(eventHandler[*api.ControlPlane]{logger, "ControlPlane", cpLister})
	pchInformer.AddEventHandler(eventHandler[*api.PostCreateHook]{logger, "PostCreateHook", pchLister})
	sif.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), cpInformer.HasSynced, pchInformer.HasSynced) {
		logger.Error(nil, "Failed to sync informer caches")
		os.Exit(20)
	}
	<-ctx.Done()
}

type mrObject interface {
	metav1.Object
	runtime.Object
}

type GenericLister[ObjectType mrObject] interface {
	// List lists all PostCreateHooks in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []ObjectType, err error)
	// Get retrieves the PostCreateHook from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (ObjectType, error)
}

type eventHandler[ObjectType mrObject] struct {
	logger klog.Logger
	kind   string
	lister GenericLister[ObjectType]
}

func (eh eventHandler[ObjectType]) OnAdd(obj any, isInitial bool) {
	mrObj := obj.(mrObject)
	eh.logger.V(2).Info("Notified of add", "kind", eh.kind, "name", mrObj.GetName())
	fromCache, err := eh.lister.Get(mrObj.GetName())
	if err != nil {
		eh.logger.Error(err, "Failed to Get object from lister", "kind", eh.kind, "name", mrObj.GetName())
	}
	if fromCache.GetName() != mrObj.GetName() {
		eh.logger.Error(nil, "Lister returned objecgt of different name", "kind", eh.kind, "name", mrObj.GetName(), "nameFromLister", fromCache.GetName())
	}
}

func (eh eventHandler[ObjectType]) OnUpdate(oldObj, obj any) {
	mrObj := obj.(mrObject)
	eh.logger.V(4).Info("Notified of update", "kind", eh.kind, "name", mrObj.GetName())
}

func (eh eventHandler[ObjectType]) OnDelete(obj any) {
	if dfsu, is := obj.(cache.DeletedFinalStateUnknown); is {
		obj = dfsu.Obj
	}
	mrObj := obj.(mrObject)
	eh.logger.V(2).Info("Notified of delete", "kind", eh.kind, "name", mrObj.GetName())
}
