package scheduler

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
)

type PodInfo struct {
	Pod        *v1.Pod
	PodRequest *ResourceQuantity
}

func NewPodInfo(pod *v1.Pod) *PodInfo {
	pi := &PodInfo{
		Pod:        pod,
		PodRequest: GetPodRequest(pod),
	}

	return pi
}

func (pi *PodInfo) Name() string {
	return fmt.Sprintf("%s/%s", pi.Pod.Namespace, pi.Pod.Name)
}

func (pi *PodInfo) NodeName() string {
	return pi.Pod.Spec.NodeName
}

func GetPodRequest(pod *v1.Pod) *ResourceQuantity {
	request := NewResourceQuantity()

	for _, container := range pod.Spec.Containers {
		if cpuReq, exists := container.Resources.Requests[v1.ResourceCPU]; exists {
			request.CPU.Add(cpuReq)
		}
		if memReq, exists := container.Resources.Requests[v1.ResourceMemory]; exists {
			request.Memory.Add(memReq)
		}
		if gpuReq, exists := container.Resources.Requests[ResourceNvidiaGPU]; exists {
			request.GPU.Add(gpuReq)
		}
	}

	return request
}
