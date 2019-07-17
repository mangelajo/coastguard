package main

import (
    "flag"
    "sync"

    "github.com/submariner-io/submariner/pkg/signals"
    "k8s.io/klog"

    "github.com/submariner-io/coastguard/pkg/clusters/fileDiscovery"
    "github.com/submariner-io/coastguard/pkg/coastguard/controller"
)

type SubmarinerRouteControllerSpecification struct {
    ClusterID string
    Namespace string
}

func main() {
    klog.InitFlags(nil)
    flag.Parse()

    klog.V(2).Info("Starting coastguard-network-policy-sync")

    // set up signals so we handle the first shutdown signal gracefully
    stopCh := signals.SetupSignalHandler()

    var wg sync.WaitGroup

    wg.Add(1)

    go func() {
        defer wg.Done()

        coastGuardController := controller.New(stopCh)
        fileDiscovery.Start(coastGuardController)
        if err := coastGuardController.Run(); err != nil {
            klog.Fatal("Error running the coastguard controller: %s", err)
        }

    }()

    wg.Wait()
    klog.Fatal("All controllers stopped or exited. Stopping main loop")
}


