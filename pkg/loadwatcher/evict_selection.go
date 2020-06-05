package loadwatcher

import (
	"k8s.io/api/core/v1"
	"k8s.io/klog"
	"sort"
)

type PodCandidateSet []PodCandidate

func (s PodCandidateSet) Len() int {
	return len(s)
}

func (s PodCandidateSet) Less(i, j int) bool {
	return s[i].Score < s[j].Score
}

func (s PodCandidateSet) Swap(i, j int) {
	x := s[i]
	s[i] = s[j]
	s[j] = x
}

type PodCandidate struct {
	Pod   *v1.Pod
	Score int
}

func PodCandidateSetFromPodList(l *v1.PodList) PodCandidateSet {
	s := make(PodCandidateSet, len(l.Items))

	for i := range l.Items {
		s[i] = PodCandidate{
			Pod:   &l.Items[i],
			Score: 0,
		}
	}

	return s
}

func (s PodCandidateSet) scoreByQOSClass() {
	for i := range s {
		switch s[i].Pod.Status.QOSClass {
		case v1.PodQOSBestEffort:
			s[i].Score += 200
		case v1.PodQOSBurstable:
			s[i].Score += 100
		}
	}
}

func (s PodCandidateSet) scoreByOwnerType() {
	for i := range s {
		// do not evict Pods without owner; these will probably not be re-scheduled if evicted
		if len(s[i].Pod.OwnerReferences) == 0 {
			s[i].Score -= 1000
		}

		for j := range s[i].Pod.OwnerReferences {
			o := &s[i].Pod.OwnerReferences[j]

			switch o.Kind {
			case "ReplicaSet":
				s[i].Score += 100
			case "StatefulSet":
				s[i].Score -= 1000
			case "DaemonSet":
				s[i].Score -= 1000
			}
		}
	}
}

func (s PodCandidateSet) scoreByCriticality() {
	for i := range s {
		if s[i].Pod.Namespace == "kube-system" {
			s[i].Score -= 1000
		}

		switch s[i].Pod.Spec.PriorityClassName {
		case "system-cluster-critical":
			s[i].Score -= 1000
		case "system-node-critical":
			s[i].Score -= 1000
		}

		if _, ok := s[i].Pod.Annotations["scheduler.alpha.kubernetes.io/critical-pod"]; ok {
			s[i].Score -= 1000
		}
	}
}

func (s PodCandidateSet) SelectPodForEviction() *v1.Pod {
	s.scoreByQOSClass()
	s.scoreByOwnerType()
	s.scoreByCriticality()

	sort.Stable(sort.Reverse(s))

	for i := range s {
		klog.Infof("eviction candidate: %s/%s (score of %d)", s[i].Pod.Namespace, s[i].Pod.Name, s[i].Score)
	}

	for i := range s {
		if s[i].Score < 0 {
			continue
		}

		klog.Infof("selected candidate: %s/%s (score of %d)", s[i].Pod.Namespace, s[i].Pod.Name, s[i].Score)
		return s[i].Pod
	}

	return nil
}
