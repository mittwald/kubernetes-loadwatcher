package loadwatcher

import (
	"context"
	"fmt"
	"github.com/mittwald/kubernetes-loadwatcher/pkg/jsonpatch"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

func (t *Tainter) IsNodeTainted(ctx context.Context) (bool, error) {
	node, err := t.client.CoreV1().Nodes().Get(ctx, t.nodeName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	for i := range node.Spec.Taints {
		if node.Spec.Taints[i].Key == TaintKey {
			return true, nil
		}
	}

	return false, nil
}

func (t *Tainter) TaintNode(ctx context.Context, evt LoadThresholdEvent) error {
	node, err := t.client.CoreV1().Nodes().Get(ctx, t.nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	nodeCopy := node.DeepCopy()

	if nodeCopy.Spec.Taints == nil {
		nodeCopy.Spec.Taints = make([]v1.Taint, 0, 1)
	}

	for i := range nodeCopy.Spec.Taints {
		if nodeCopy.Spec.Taints[i].Key == TaintKey {
			klog.Infof("wanted to taint node %s, but taint already exists", nodeCopy.Name)
			return nil
		}
	}

	nodeCopy.Spec.Taints = append(nodeCopy.Spec.Taints, v1.Taint{
		Key:    TaintKey,
		Value:  "true",
		Effect: v1.TaintEffectPreferNoSchedule,
	})

	_, err = t.client.CoreV1().Nodes().Update(ctx, nodeCopy, metav1.UpdateOptions{})

	t.recorder.Eventf(t.nodeRef, v1.EventTypeWarning, "LoadThresholdExceeded", "load5 on node was %.2f; exceeded threshold of %.2f. tainting node", evt.Load5, evt.LoadThreshold)

	if err != nil {
		t.recorder.Eventf(t.nodeRef, v1.EventTypeWarning, "NodePatchError", "could not patch node: %s", err.Error())
		return err
	}

	return nil
}

func (t *Tainter) UntaintNode(ctx context.Context, evt LoadThresholdEvent) error {
	node, err := t.client.CoreV1().Nodes().Get(ctx, t.nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	taintIndex := -1

	for i, t := range node.Spec.Taints {
		if t.Key == TaintKey {
			taintIndex = i
			break
		}
	}

	if taintIndex == -1 {
		klog.Infof("wanted to remove taint from node %s, but taint was already gone", node.Name)
		return nil
	}

	t.recorder.Eventf(t.nodeRef, v1.EventTypeNormal, "LoadThresholdDeceeded", "load15 on node was %.2f; deceeded threshold of %.2f. untainting node", evt.Load15, evt.LoadThreshold)

	_, err = t.client.CoreV1().Nodes().Patch(ctx, t.nodeName, types.JSONPatchType, jsonpatch.PatchList{{
		Op: "test",
		Path: fmt.Sprintf("/spec/taints/%d/key", taintIndex),
		Value: TaintKey,
	}, {
		Op:   "remove",
		Path: fmt.Sprintf("/spec/taints/%d", taintIndex),
		Value: "",
	}}.ToJSON(), metav1.PatchOptions{})

	if err != nil {
		t.recorder.Eventf(t.nodeRef, v1.EventTypeWarning, "NodePatchError", "could not patch node: %s", err.Error())
		return err
	}

	return nil
}
