package scheduler

import (
    v1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
)

type NodeInfo struct {
    node        *v1.Node
    Pods        map[string]*PodInfo
    Allocatable *ResourceQuantity
    Requests    *ResourceQuantity
}

func NewNodeInfo(node *v1.Node, pods map[string]*PodInfo) *NodeInfo {
    requests := NewResourceQuantity()
    for _, pod := range pods {
       requests.Add(*pod.PodRequest)
    }

    return &NodeInfo{
       node:     node,
       Pods:     pods,
       Requests: requests,
       Allocatable: &ResourceQuantity{
          CPU:    resource.NewQuantity(node.Status.Allocatable.Cpu().Value(), resource.DecimalSI),
          Memory: resource.NewQuantity(node.Status.Allocatable.Memory().Value(), resource.DecimalSI),
          GPU:    resource.NewQuantity(node.Status.Allocatable.Name(ResourceNvidiaGPU, resource.DecimalSI).Value(), resource.DecimalSI),
       },
    }
}

func (ni *NodeInfo) Name() string {
    return ni.node.Name
}

func (ni *NodeInfo) HasPod(podName string) bool {
	_, ok := ni.Pods[podName]
	return ok	
}

func (ni *NodeInfo) Free() *ResourceQuantity {
	free := NewResourceQuantity()
	free.Add(*ni.Allocatable)
	free.Sub(*ni.Requests)
	return free
}
