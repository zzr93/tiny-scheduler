package scheduler

type NodeScore struct {
	Name  string
	Score int64
}

type ScoreNodePlugin func(nodeInfos []*NodeInfo, pod *PodInfo) []NodeScore

// EqualScorePlugin 等分插件，每个节点得分相同，仅作为示例
func EqualScorePlugin(nodeInfos []*NodeInfo, podInfo *PodInfo) []NodeScore {
	var scores []NodeScore

	for _, nodeInfo := range nodeInfos {
		score := NodeScore{
			Name:  nodeInfo.Name(),
			Score: 1000,
		}
		scores = append(scores, score)
	}

	return scores
}
