package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/mittwald/kubernetes-loadwatcher/pkg/config"
	"github.com/mittwald/kubernetes-loadwatcher/pkg/loadwatcher"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var f config.StartupFlags

	flag.StringVar(&f.KubeConfig, "kubeconfig", "", "file path to kubeconfig")
	flag.IntVar(&f.TaintThreshold, "taint-threshold", 0, "load threshold value (set to 0 for automatic detection)")
	flag.IntVar(&f.EvictThreshold, "evict-threshold", 0, "load threshold value (set to 0 for automatic detection)")
	flag.StringVar(&f.EvictBackoff, "evict-backoff", "10m", "time to wait between evicting Pods")
	flag.StringVar(&f.NodeName, "node-name", "", "current node name")
	flag.Parse()

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

	closeChan := make(chan struct{})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		s := <-sigChan

		glog.Infof("received signal %s", s)

		close(closeChan)
	}()

	isTainted, err := t.IsNodeTainted()
	if err != nil {
		panic(err)
	}

	w.SetAsHigh(isTainted)

	exc, dec, errs := w.Run(closeChan)
	for {
		select {
		case evt, ok := <-exc:
			if !ok {
				glog.Infof("exceedance channel closed; stopping")
				return
			}

			glog.Infof("load5 exceeded threshold, load5=%f load15=%f", evt.Load5, evt.Load15)

			if err := t.TaintNode(evt); err != nil {
				glog.Errorf("error while tainting node: %s", err.Error())
			}

			if _, err := e.EvictPod(evt); err != nil {
				glog.Errorf("error while evicting pod: %s", err.Error())
			}
		case evt, ok := <-dec:
			if !ok {
				glog.Infof("deceedance channel closed; stopping")
				return
			}

			glog.Infof("load15 deceeded threshold, load5=%f load15=%f", evt.Load5, evt.Load15)

			if err := t.UntaintNode(evt); err != nil {
				glog.Errorf("error while removing taint from node: %s", err.Error())
			}
		case err := <-errs:
			glog.Errorf("error while polling for status updates: %s", err.Error())
		}
	}
}

func loadKubernetesConfig(f config.StartupFlags) (*rest.Config, error) {
	if f.KubeConfig == "" {
		return rest.InClusterConfig()
	}

	return clientcmd.BuildConfigFromFlags("", f.KubeConfig)
}