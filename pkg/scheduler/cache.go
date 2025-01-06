package scheduler

import (
	"sync"
	"time"
)

type PodNodeCache struct {
	mu sync.RWMutex

	nodePodMap map[string]map[string]podInfoCache
	podNodeMap map[string]string
}

type podInfoCache struct {
	timeStamp time.Time
	podInfo   *PodInfo
}

func (pnc *podInfoCache) IsExpired() bool {
	return time.Since(pnc.timeStamp) > 10*time.Second
}

func NewPodNodeCache() *PodNodeCache {
	pnc := &PodNodeCache{
		nodePodMap: make(map[string]map[string]podInfoCache),
		podNodeMap: make(map[string]string),
	}
	pnc.cleanExpiredCache()
	return pnc
}

func (pnc *PodNodeCache) cleanExpiredCache() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			for _, pods := range pnc.nodePodMap {
				for pod, cache := range pods {
					if cache.IsExpired() {
						pnc.ForgetPod(pod)
					}
				}
			}
		}
	}()
}

func (pnc *PodNodeCache) AddPod(nodeName string, podInfo *PodInfo) {
	pnc.mu.Lock()
	defer pnc.mu.Unlock()

	if _, exist := pnc.nodePodMap[nodeName]; !exist {
		pnc.nodePodMap[nodeName] = make(map[string]podInfoCache)
	}
	pnc.nodePodMap[nodeName][podInfo.Name()] = podInfoCache{
		timeStamp: time.Now(),
		podInfo:   podInfo,
	}
	pnc.podNodeMap[podInfo.Name()] = nodeName
}

func (pnc *PodNodeCache) ForgetPod(pod string) {
	pnc.mu.Lock()
	defer pnc.mu.Unlock()

	if nodeName, exist := pnc.podNodeMap[pod]; exist {
		delete(pnc.nodePodMap[nodeName], pod)
	}
	delete(pnc.podNodeMap, pod)
}

func (pnc *PodNodeCache) HasPod(pod string) bool {
	pnc.mu.RLock()
	defer pnc.mu.RUnlock()

	_, exist := pnc.podNodeMap[pod]
	return exist
}

func (pnc *PodNodeCache) RangeNodePods(nodeName string, fn func(pod string, podInfo *PodInfo)) {
	pnc.mu.RLock()
	defer pnc.mu.RUnlock()

	for pod, cache := range pnc.nodePodMap[nodeName] {
		fn(pod, cache.podInfo)
	}
}
