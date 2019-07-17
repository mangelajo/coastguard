package clusterwatcher

import (
    "time"

    v1 "k8s.io/api/core/v1"
    "k8s.io/client-go/informers"
    v12 "k8s.io/client-go/informers/core/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/cache"
    "k8s.io/klog"
)

type ClusterWatcher struct {
    stopCh <-chan struct{}
    clusterID string
    pods [] v1.Pod
    clientSet *kubernetes.Clientset

    informerFactory informers.SharedInformerFactory
    podInformer v12.PodInformer

}

func NewClusterWatcher(clusterID string, kubeConfig *rest.Config, stopCh <-chan struct{}) (*ClusterWatcher, error){

    clientSet, err := kubernetes.NewForConfig(kubeConfig)
    if err != nil {
        klog.Fatalf("Error creating clientset for cluster %s: %s", clusterID, err.Error())
    }

    clusterWatcher := &ClusterWatcher {
        clusterID: clusterID,
        clientSet: clientSet,
        stopCh: stopCh,
    }

    err = clusterWatcher.init()

    return clusterWatcher, err
}


func (c *ClusterWatcher) init() error  {


    factory := informers.NewSharedInformerFactory(c.clientSet, time.Hour*24)
    informer := factory.Core().V1().Pods().Informer()
    informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: c.onPodAdd,
    })

    go informer.Run(c.stopCh)

    if !cache.WaitForCacheSync(c.stopCh, informer.HasSynced) {
        klog.Fatal("Timed out waiting for caches to sync")
    }


    return nil
}

func (c *ClusterWatcher) onPodAdd(obj interface{}) {
    pod := obj.(*v1.Pod)
    klog.Info("pod added in cluster ", c.clusterID, " ", pod.GetSelfLink())
}