package remotecluster

import (
    "sync"
    "time"

    v1 "k8s.io/api/core/v1"
    v1net "k8s.io/api/networking/v1"
    "k8s.io/client-go/informers"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/cache"
    "k8s.io/klog"
)


// initialWaitSeconds is the time we wait
const initialWaitSeconds = 10 * time.Second

type RemoteCluster struct {

    // notificationMutex is used to block incoming events during an
    // OnSyncDone event being sent back to the controller
    notificationMutex     sync.Mutex
    ClusterID             string
    ClientSet             *kubernetes.Clientset
    podInformer           cache.SharedIndexInformer
    networkPolicyInformer cache.SharedIndexInformer

    eventsLastPeriod int

    OnAddFunc func(resourceWatcher *RemoteCluster, obj interface{})
    OnUpdateFunc func(resourceWatcher *RemoteCluster, oldObj interface{}, newObj interface{})
    OnDeleteFunc func(resourceWatcher *RemoteCluster, obj interface{})
    OnSyncDoneFunc func(resourceWatcher *RemoteCluster)

}

func New(clusterID string, kubeConfig *rest.Config) (*RemoteCluster, error){

    clientSet, err := kubernetes.NewForConfig(kubeConfig)
    if err != nil {
        klog.Errorf("Error creating clientset for cluster %s: %s", clusterID, err.Error())
        return nil, err
    }

    factory := informers.NewSharedInformerFactory(clientSet, time.Hour*24)
    podInformer := factory.Core().V1().Pods().Informer()
    networkPolicyInformer := factory.Networking().V1().NetworkPolicies().Informer()

    resourceWatcher := &RemoteCluster{
        ClusterID:             clusterID,
        ClientSet:             clientSet,
        podInformer:           podInformer,
        networkPolicyInformer: networkPolicyInformer,
    }

    // keeping the wrapper to not expose onAdd / onDelete / onUpdate handlers
    handlers := cache.ResourceEventHandlerFuncs{
        AddFunc: resourceWatcher.onAdd,
        DeleteFunc: resourceWatcher.onDelete,
        UpdateFunc: resourceWatcher.onUpdate,
    }

    podInformer.AddEventHandler(handlers)
    networkPolicyInformer.AddEventHandler(handlers)

    return resourceWatcher, nil
}

func (rw *RemoteCluster) HasSynced() bool {
    return rw.podInformer.HasSynced() &&
        rw.networkPolicyInformer.HasSynced()
}

func (rw *RemoteCluster) Run(stopCh <-chan struct{}) {
    go rw.podInformer.Run(stopCh)
    go rw.networkPolicyInformer.Run(stopCh)

    if !cache.WaitForCacheSync(stopCh, rw.podInformer.HasSynced) {
        klog.Warning("Timed out waiting for pod informer to sync")
    }

    if !cache.WaitForCacheSync(stopCh, rw.networkPolicyInformer.HasSynced) {
        klog.Warning("Timed out waiting for NetworkPolicy informer to sync")
    }

    if rw.OnSyncDoneFunc != nil {
        rw.notificationMutex.Lock()
        rw.OnSyncDoneFunc(rw)
        rw.notificationMutex.Unlock()
    }

}

func (rw *RemoteCluster) GetPods() [] interface{} {
    return rw.podInformer.GetStore().List()
}
func (rw *RemoteCluster) GetNetworkPolicies() [] interface{} {
    return rw.networkPolicyInformer.GetStore().List()
}


func (rw *RemoteCluster) onAdd(obj interface{}) {

    rw.notificationMutex.Lock()
    defer rw.notificationMutex.Unlock()

    if pod, ok := obj.(*v1.Pod); ok {
        klog.Infof("Pod discovered in %s: %s", rw.ClusterID, pod.GetSelfLink())
    } else if np, ok := obj.(*v1net.NetworkPolicy); ok {
        klog.Infof("NetworkPolicy discovered in %s: %s", rw.ClusterID, np.GetSelfLink())
    }
    if rw.OnAddFunc != nil {
        rw.OnAddFunc(rw, obj)
    }
}

func (rw *RemoteCluster) onDelete(obj interface{}) {

    rw.notificationMutex.Lock()
    defer rw.notificationMutex.Unlock()

    if pod, ok := obj.(*v1.Pod); ok {
        klog.Infof("Pod deleted in %s: %s", rw.ClusterID, pod.GetSelfLink())
    } else if np, ok := obj.(*v1net.NetworkPolicy); ok {
        klog.Infof("NetworkPolicy deleted in %s: %s", rw.ClusterID, np.GetSelfLink())
    }
    if rw.OnDeleteFunc != nil {
        rw.OnDeleteFunc(rw, obj)
    }
}

func (rw *RemoteCluster) onUpdate(oldObj interface{}, newObj interface{}) {

    rw.notificationMutex.Lock()
    defer rw.notificationMutex.Unlock()

    if pod, ok := newObj.(*v1.Pod); ok {
        klog.Infof("Pod updated in %s: %s", rw.ClusterID, pod.GetSelfLink())
    } else if np, ok := newObj.(*v1net.NetworkPolicy); ok {
        klog.Infof("NetworkPolicy updated in %s: %s", rw.ClusterID, np.GetSelfLink())
    }
    if rw.OnUpdateFunc != nil {
        rw.OnUpdateFunc(rw, oldObj, newObj)
    }
}

