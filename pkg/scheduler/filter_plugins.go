package scheduler

type FilterNodePlugin func(nodeInfos []*NodeInfo, podInfo *PodInfo) []*NodeInfo

func ResourceFilter(nodeInfos []*NodeInfo, podInfo *PodInfo) []*NodeInfo {
	var filteredNodes []*NodeInfo

	for _, nodeInfo := range nodeInfos {
		if nodeInfo.Free().Accommodate(*podInfo.PodRequest) {
			filteredNodes = append(filteredNodes, nodeInfo)
		}
	}
	return filteredNodes
}