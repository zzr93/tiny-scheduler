package scheduler

import "k8s.io/apimachinery/pkg/api/resource"

const (
	ResourceNvidiaGPU = "nvidia.com/gpu"
)

type ResourceQuantity struct {
    CPU    *resource.Quantity
    Memory *resource.Quantity
    GPU    *resource.Quantity
}

func NewResourceQuantity() *ResourceQuantity {
    return &ResourceQuantity{
       CPU:    resource.NewQuantity(0, resource.DecimalSI),
       Memory: resource.NewQuantity(0, resource.DecimalSI),
       GPU:    resource.NewQuantity(0, resource.DecimalSI),
    }
}

func NewResourceQuantityFromQuantity(cpu, memory, gpu resource.Quantity) *ResourceQuantity {
    return &ResourceQuantity{
       CPU:    &cpu,
       Memory: &memory,
       GPU:    &gpu,
    }
}

func (r *ResourceQuantity) Add(request ResourceQuantity) {
    if r == nil {
       return
    }

    r.CPU.Add(*request.CPU)
    r.Memory.Add(*request.Memory)
    r.GPU.Add(*request.GPU)
}

func (r *ResourceQuantity) Sub(request ResourceQuantity) {
    if r == nil {
       return
    }

    r.CPU.Sub(*request.CPU)
    r.Memory.Sub(*request.Memory)
    r.GPU.Sub(*request.GPU)
}

func (r *ResourceQuantity) Accommodate(rq ResourceQuantity) bool {
    if r == nil {
       return false
    }

    if r.CPU.Cmp(*rq.CPU) == -1 {
       return false
    }
    if r.Memory.Cmp(*rq.Memory) == -1 {
       return false
    }
    if r.GPU.Cmp(*rq.GPU) == -1 {
       return false
    }

    return true
}

func (r *ResourceQuantity) Positive() bool {
    if r == nil {
       return false
    }

    return r.CPU.Cmp(*resource.NewQuantity(0, resource.DecimalSI)) > 0 &&
       r.Memory.Cmp(*resource.NewQuantity(0, resource.DecimalSI)) > 0 &&
       r.GPU.Cmp(*resource.NewQuantity(0, resource.DecimalSI)) > 0
}