package kube

import (
	"errors"
	"path/filepath"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var defaultClient *kubernetes.Clientset
var stopper = make(chan struct{})

func InitKubeClient() error {
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return err
		}
	}

	defaultClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	return InitInformers(stopper, WithNode(), WithPod())
}

func DefaultKubeClient() *kubernetes.Clientset {
	return defaultClient
}

func NodeLister() v1.NodeLister {
	return nodeLister
}

func PodLister() v1.PodLister {
	return podLister
}

var factory informers.SharedInformerFactory

var nodeInformer cache.SharedIndexInformer
var nodeLister v1.NodeLister

var podInformer cache.SharedIndexInformer
var podLister v1.PodLister

type InformerOption func() cache.SharedIndexInformer

func WithNode() InformerOption {
	return func() cache.SharedIndexInformer {
		nodeInformer = factory.Core().V1().Nodes().Informer()
		nodeLister = factory.Core().V1().Nodes().Lister()
		return nodeInformer
	}
}

func WithPod() InformerOption {
	return func() cache.SharedIndexInformer {
		podInformer = factory.Core().V1().Pods().Informer()
		podLister = factory.Core().V1().Pods().Lister()
		return podInformer
	}
}

func InitInformers(stopper <-chan struct{}, opts ...InformerOption) error {
	factory = informers.NewSharedInformerFactory(DefaultKubeClient(), time.Hour)

	infos := make([]cache.SharedIndexInformer, len(opts))
	for i := range opts {
		currentInformer := opts[i]()
		infos[i] = currentInformer
		go currentInformer.Run(stopper)
	}

	for i := range infos {
		if !cache.WaitForCacheSync(stopper, infos[i].HasSynced) {
			return errors.New("failed to wait for cache to sync")
		}
	}

	return nil
}
