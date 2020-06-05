package loadwatcher

import (
	"context"
	"k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/klog"
	"time"
)

// CanEvict determines if the evicter can now evict a Pod at this time, or if it
// is still in its back-off period.
func (e *Evicter) CanEvict() bool {
	if e.lastEviction.IsZero() {
		return true
	}

	return time.Now().Sub(e.lastEviction) > e.backoff
}

// EvictPod tries to pick a suitable Pod for eviction and evict it.
func (e *Evicter) EvictPod(ctx context.Context, evt LoadThresholdEvent) (bool, error) {
	if evt.Load15 < e.threshold {
		return false, nil
	}

	if !e.CanEvict() {
		klog.Infof("eviction threshold exceeded; still in back-off")
		return false, nil
	}

	klog.Infof("searching for pod to evict")

	fieldSelector := fields.OneTermEqualSelector("spec.nodeName", e.nodeName)

	podsOnNode, err := e.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})

	if err != nil {
		return false, err
	}

	candidates := PodCandidateSetFromPodList(podsOnNode)
	podToEvict := candidates.SelectPodForEviction()

	if podToEvict == nil {
		e.recorder.Eventf(e.nodeRef, v1.EventTypeWarning, "NoPodToEvict", "wanted to evict Pod, but no suitable candidate found")
		return false, nil
	}

	eviction := v1beta1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podToEvict.ObjectMeta.Name,
			Namespace: podToEvict.ObjectMeta.Namespace,
		},
	}

	klog.Infof("eviction: %+v", eviction)

	e.lastEviction = time.Now()

	e.recorder.Eventf(podToEvict, v1.EventTypeWarning, "EvictHighLoad", "evicting pod due to high load on node load15=%.2f threshold=%.2f", evt.Load15, evt.LoadThreshold)
	e.recorder.Eventf(e.nodeRef, v1.EventTypeWarning, "EvictHighLoad", "evicting pod due to high load on node load15=%.2f threshold=%.2f", evt.Load15, evt.LoadThreshold)

	err = e.client.CoreV1().Pods(podToEvict.Namespace).Evict(ctx, &eviction)
	return true, err
}
