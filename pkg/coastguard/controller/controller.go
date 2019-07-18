package controller

import (
    "net/http"
    "time"

    "github.com/submariner-io/submariner/pkg/federate"
    "k8s.io/client-go/rest"
    "k8s.io/klog"

    "github.com/submariner-io/coastguard/pkg/remotecluster"
)

const initialSyncTimeout = 60 * time.Second

type CoastguardController struct {
    federate.ClusterEventHandler
    remoteClusters map[string] *remotecluster.RemoteCluster
    syncedClusters map[string] *remotecluster.RemoteCluster

    stopCh         <-chan struct{}

}


func New(stopCh <-chan struct{}) *CoastguardController {

    return &CoastguardController{
        remoteClusters:      make(map[string] *remotecluster.RemoteCluster),
        syncedClusters:      make(map[string] *remotecluster.RemoteCluster),
        stopCh: stopCh,
    }
}

func (c* CoastguardController) OnAdd(clusterID string, kubeConfig *rest.Config) {
    klog.Infof("adding cluster: %s", clusterID)

    if cw, err := remotecluster.New(clusterID, kubeConfig); err == nil {
        c.remoteClusters[clusterID] = cw
        cw.OnSyncDoneFunc = c.onClusterFinishedSyncing
        cw.Run(c.stopCh)
    } else {
        klog.Errorf("There was an issue adding the remote cluster %s: %s", clusterID, err.Error())
    }
}

func (c *CoastguardController) onClusterFinishedSyncing(cluster *remotecluster.RemoteCluster) {
    klog.Infof("Cluster %s finished syncing", cluster.ClusterID)
    c.syncedClusters[cluster.ClusterID] = cluster
}

func (c* CoastguardController) OnRemove(clusterID string) {
    klog.Infof("removing cluster: %s", clusterID)
    klog.Fatalf("Not implemented yet")
}

func (c* CoastguardController) OnUpdate(clusterID string, kubeConfig *rest.Config) {
    klog.Infof("updating cluster: %s", clusterID)
    klog.Fatalf("Not implemented yet")
}


func (c* CoastguardController) Run() error {

    go serveHealthz(":8080")

    go c.processLoop()

    <- c.stopCh
    return nil
}

func (c *CoastguardController) processLoop() {

    c.startBarrier()

    for {
        time.Sleep(1*time.Second)
    }
}

// startBarrier waits for at least one cluster to show up, and then
// for all registered clusters to be synchronized with the local cache
func (c *CoastguardController) startBarrier() {
    klog.Info("Waiting for clusters to coastguard")
    for {
        if len(c.remoteClusters) > 0 {
            break
        }
        time.Sleep(1 * time.Second)
    }
    time.Sleep(5 * time.Second)
    start := time.Now()
    for {
        if len(c.remoteClusters) == len(c.syncedClusters) {
            klog.Infof("Initial sync finished for %d clusters", len(c.remoteClusters))
            break
        }
        if time.Since(start) >= initialSyncTimeout {
            klog.Warningf("Timeout: initial sync finished for %d clusters, from a %d total, continuing anyway.")
            break
        }

        klog.Infof("Waiting for sync with remote clusters (%d synced of %d)", len(c.syncedClusters), len(c.remoteClusters))
        time.Sleep(5 * time.Second)
    }
}


func serveHealthz(address string) {
    http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("OK"))
    })

    klog.Fatal(http.ListenAndServe(address, nil))
}