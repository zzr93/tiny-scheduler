package scheduler

import (
	"github.com/golang/glog"
	"github.com/zzr93/tiny-scheduler/pkg/kube"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sort"
)

type Queue struct {
	// todo 调度队列可以考虑使用优先队列
	list []*PodInfo
}

func NewQueue() *Queue {
	return &Queue{
		list: make([]*PodInfo, 0),
	}
}

// Update fetch data from db to scheduling list
func (q *Queue) Update() {
	pods, err := kube.PodLister().List(labels.NewSelector())
	if err != nil {
		glog.Errorf("Update queue error while listing pods: %v", err)
		return
	}

	var schedulingPodInfos []*PodInfo
	for _, pod := range pods {
		if pod.Spec.SchedulerName == GetSchedulerName() && pod.Status.Phase == v1.PodPending && pod.Spec.NodeName == "" {
			schedulingPodInfos = append(schedulingPodInfos, NewPodInfo(pod))
		}
	}

	sort.Slice(schedulingPodInfos, func(i, j int) bool {
		return schedulingPodInfos[i].Pod.CreationTimestamp.Before(&schedulingPodInfos[j].Pod.CreationTimestamp)
	})

	q.list = schedulingPodInfos
}
