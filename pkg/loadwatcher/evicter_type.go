package loadwatcher

import (
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"time"
)

type Evicter struct {
	client       kubernetes.Interface
	threshold    float64
	nodeName     string
	nodeRef      *v1.ObjectReference
	recorder     record.EventRecorder
	backoff      time.Duration
	lastEviction time.Time
}

func NewEvicter(client kubernetes.Interface, threshold int, nodeName string, backoff string) (*Evicter, error) {
	if threshold == 0 {
		cpuCount, err := determineCPUCount()
		if err != nil {
			return nil, err
		}

		threshold = int(cpuCount) * 4
	}

	backoffDuration, err := time.ParseDuration(backoff)
	if err != nil {
		return nil, err
	}

	b := record.NewBroadcaster()
	b.StartLogging(glog.Infof)
	b.StartRecordingToSink(&typedv1.EventSinkImpl{
		Interface: client.CoreV1().Events(""),
	})

	r := b.NewRecorder(scheme.Scheme, v1.EventSource{Host: nodeName, Component: ComponentName + "/evicter"})

	nodeRef := &v1.ObjectReference{
		Kind:      "Node",
		Name:      nodeName,
		UID:       types.UID(nodeName),
		Namespace: "",
	}

	return &Evicter{
		client:    client,
		threshold: float64(threshold),
		nodeName:  nodeName,
		nodeRef:   nodeRef,
		recorder:  r,
		backoff:   backoffDuration,
	}, nil
}
