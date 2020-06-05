package main

import (
	"context"
	"flag"
	"github.com/mittwald/kubernetes-loadwatcher/pkg/config"
	"github.com/mittwald/kubernetes-loadwatcher/pkg/loadwatcher"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var f config.StartupFlags

	klog.InitFlags(nil)

	flag.StringVar(&f.KubeConfig, "kubeconfig", "", "file path to kubeconfig")
	flag.IntVar(&f.TaintThreshold, "taint-threshold", 0, "load threshold value (set to 0 for automatic detection)")
	flag.IntVar(&f.EvictThreshold, "evict-threshold", 0, "load threshold value (set to 0 for automatic detection)")
	flag.StringVar(&f.EvictBackoff, "evict-backoff", "10m", "time to wait between evicting Pods")
	flag.StringVar(&f.NodeName, "node-name", "", "current node name")
	flag.Parse()

	if f.NodeName == "" {
		panic("-node-name not set")
	}

	cfg, err := loadKubernetesConfig(f)
	if err != nil {
		panic(err)
	}

	c, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	w, err := loadwatcher.NewWatcher(f.TaintThreshold)
	if err != nil {
		panic(err)
	}

	t, err := loadwatcher.NewTainter(c, f.NodeName)
	if err != nil {
		panic(err)
	}

	e, err := loadwatcher.NewEvicter(c, f.EvictThreshold, f.NodeName, f.EvictBackoff)
	if err != nil {
		panic(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	ctx, cancelFn := context.WithCancel(context.Background())

	go func() {
		s := <-sigChan

		klog.Infof("received signal %s", s)

		cancelFn()
	}()

	isTainted, err := t.IsNodeTainted(ctx)
	if err != nil {
		panic(err)
	}

	w.SetAsHigh(isTainted)

	exc, dec, errs := w.Run(ctx)
	for {
		select {
		case evt, ok := <-exc:
			if !ok {
				klog.Infof("exceedance channel closed; stopping")
				return
			}

			klog.Infof("load5 exceeded threshold, load5=%f load15=%f", evt.Load5, evt.Load15)

			if err := t.TaintNode(ctx, evt); err != nil {
				klog.Errorf("error while tainting node: %s", err.Error())
			}

			if _, err := e.EvictPod(ctx, evt); err != nil {
				klog.Errorf("error while evicting pod: %s", err.Error())
			}
		case evt, ok := <-dec:
			if !ok {
				klog.Infof("deceedance channel closed; stopping")
				return
			}

			klog.Infof("load15 deceeded threshold, load5=%f load15=%f", evt.Load5, evt.Load15)

			if err := t.UntaintNode(ctx, evt); err != nil {
				klog.Errorf("error while removing taint from node: %s", err.Error())
			}
		case err, ok := <-errs:
			if !ok {
				return
			}

			if err != nil {
				klog.Errorf("error while polling for status updates: %s", err.Error())
			}
		}
	}
}

func loadKubernetesConfig(f config.StartupFlags) (*rest.Config, error) {
	if f.KubeConfig == "" {
		return rest.InClusterConfig()
	}

	return clientcmd.BuildConfigFromFlags("", f.KubeConfig)
}
