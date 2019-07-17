package controller

import (
    "net/http"

    "github.com/submariner-io/submariner/pkg/federate"
    "k8s.io/client-go/rest"
    "k8s.io/klog"

    "github.com/submariner-io/coastguard/pkg/clusterwatcher"
)


type CoastguardController struct {
    federate.ClusterEventHandler
    clusterWatchers map[string] *clusterwatcher.ClusterWatcher
    stopCh <-chan struct{}
}



func (c* CoastguardController) OnAdd(clusterID string, kubeConfig *rest.Config) {
    klog.Info("OnAdd: %s", clusterID)


    if cw, err := clusterwatcher.NewClusterWatcher(clusterID, kubeConfig, c.stopCh); err != nil {
        c.clusterWatchers[clusterID] = cw
    }
}



func (c* CoastguardController) OnRemove(clusterID string) {
    klog.Info("OnRemove: %s", clusterID)
}

func (c* CoastguardController) OnUpdate(clusterID string, kubeConfig *rest.Config) {
    klog.Info("OnAdd: %s", clusterID)
}


func New(stopCh <-chan struct{}) *CoastguardController {
    return &CoastguardController{stopCh: stopCh}
}


func (c* CoastguardController) Run() error {


    go serveHealthz(":8080")
    <- c.stopCh
    return nil
}



func serveHealthz(address string) {
    http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("OK"))
    })

    klog.Fatal(http.ListenAndServe(address, nil))
}