package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/golang/glog"

	"github.com/zzr93/tiny-scheduler/pkg/kube"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type Config struct {
	Name string
}

type Scheduler struct {
	config *Config
	// 调度队列
	Queue *Queue
	// 假设器
	AssumeCache *PodNodeCache

	// 过滤插件
	FilterNodePlugins []FilterNodePlugin
	// 打分插件
	ScoreNodePlugins []ScoreNodePlugin
}

type ScheduleResult struct {
	// Name of the selected node.
	SuggestedHost string
	// The number of nodes out of the evaluated ones that fit the baseJob.
	FeasibleNodes int
}

var scheduler *Scheduler

func InitScheduler(config *Config) {
	scheduler = New(config)
	scheduler.Run()
}

func GetSchedulerName() string {
	if scheduler == nil {
		return ""
	}
	return scheduler.config.Name
}

// New returns a Scheduler
func New(config *Config) *Scheduler {
	return &Scheduler{
		config:      config,
		Queue:       NewQueue(),
		AssumeCache: NewPodNodeCache(),
		FilterNodePlugins: []FilterNodePlugin{
			ResourceFilter,
		},
		ScoreNodePlugins: []ScoreNodePlugin{
			EqualScorePlugin,
		},
	}
}

// Run begins watching and scheduling. It starts scheduling and blocked until the context is done.
func (sched *Scheduler) Run() {
	go sched.schedulingCycle()
}

func (sched *Scheduler) schedulingCycle() {
	for {
		sched.Queue.Update()

		for _, pod := range sched.Queue.list {
			sched.ScheduleOne(pod)
		}

		time.Sleep(1 * time.Second)
	}
}

// ScheduleOne does the entire scheduling workflow for a single baseJob. It is serialized on the scheduling algorithm's host fitting.
func (sched *Scheduler) ScheduleOne(podInfo *PodInfo) {
	if sched.skipSchedule(podInfo) {
		glog.Infof("Skip scheduling podInfo %s", podInfo.Name())
		return
	}

	scheduleResult, err := sched.schedulePod(podInfo)
	if err != nil {
		glog.Infof("Error while scheduling podInfo %s: %s", podInfo.Pod.Name, err)
		return
	}

	err = sched.bind(podInfo, scheduleResult.SuggestedHost)
	if err != nil {
		glog.Errorf("Error while binding podInfo %s: %s", podInfo.Pod.Name, err)
		return
	}
	glog.Infof("Scheduled podInfo %s to node %s", podInfo.Pod.Name, scheduleResult.SuggestedHost)
}

// skipSchedule 判断是否需要跳过调度
func (sched *Scheduler) skipSchedule(podInfo *PodInfo) bool {
	if podInfo.Pod.DeletionTimestamp != nil {
		return true
	}

	return sched.AssumeCache.HasPod(podInfo.Name())
}

// schedulePod 尝试调度pod到指定节点
// 如果成功，返回调度结果，包括调度到的节点名
// 如果失败，返回错误信息
func (sched *Scheduler) schedulePod(podInfo *PodInfo) (result ScheduleResult, err error) {
	feasibleNodes, err := sched.findNodesThatFitPod(podInfo)
	if err != nil {
		return result, err
	}

	if len(feasibleNodes) == 0 {
		return result, errors.New("no feasible nodes found")
	}

	// When only one nodeScore after predicate, just use it.
	if len(feasibleNodes) == 1 {
		return ScheduleResult{
			SuggestedHost: feasibleNodes[0].node.Name,
			FeasibleNodes: 1,
		}, nil
	}

	nodeScores := sched.prioritizeNodes(podInfo, feasibleNodes)

	nodeScore, err := sched.selectHost(nodeScores)
	if nodeScore == nil {
		return result, errors.New("no feasible nodes selected")
	}

	return ScheduleResult{
		SuggestedHost: nodeScore.Name,
		FeasibleNodes: len(feasibleNodes),
	}, err
}

// snapshotNodeInfo 获取所有节点信息
func (sched *Scheduler) snapshotNodeInfo() ([]*NodeInfo, error) {
	allNodes, err := kube.NodeLister().List(labels.NewSelector())
	if err != nil {
		return nil, err
	}

	nodePodsMap := make(map[string]map[string]*PodInfo)
	addPod := func(nodeName string, pod *PodInfo) {
		if _, ok := nodePodsMap[nodeName]; !ok {
			nodePodsMap[nodeName] = make(map[string]*PodInfo)
		}
		if _, ok := nodePodsMap[nodeName][pod.Name()]; !ok {
			nodePodsMap[nodeName][pod.Name()] = pod
		}
	}

	pods, err := kube.PodLister().List(labels.NewSelector())
	if err != nil {
		return nil, err
	}

	// 计算informer已经能看到的活跃已调度pod的资源用量
	for _, pod := range pods {
		podInfo := NewPodInfo(pod)

		nodeName := podInfo.NodeName()
		// 未分配node的pod不占用资源
		if nodeName == "" {
			continue
		}

		addPod(nodeName, podInfo)
	}

	var nodeInfos []*NodeInfo
	for _, node := range allNodes {
		// 计算 assume 消耗的部分，这部分可能是lister感知不到的
		sched.AssumeCache.RangeNodePods(node.Name, func(podName string, podInfo *PodInfo) {
			addPod(node.Name, podInfo)
		})

		nodeInfos = append(nodeInfos, NewNodeInfo(node, nodePodsMap[node.Name]))
	}
	return nodeInfos, nil
}

// findNodesThatFitPod 找到所有可以调度到指定pod的节点
func (sched *Scheduler) findNodesThatFitPod(podInfo *PodInfo) ([]*NodeInfo, error) {
	allNodes, err := sched.snapshotNodeInfo()
	if err != nil {
		return nil, err
	}

	feasibleNodes := allNodes
	for _, filter := range sched.FilterNodePlugins {
		feasibleNodes = filter(feasibleNodes, podInfo)
	}

	return feasibleNodes, nil
}

// prioritizeNodes 对节点进行打分
func (sched *Scheduler) prioritizeNodes(podInfo *PodInfo, nodes []*NodeInfo) []*NodeScore {

	var maxScore int64
	allNodeScores := make(map[string]*NodeScore, len(nodes))

	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := range sched.ScoreNodePlugins {
		wg.Add(1)
		go func(extIndex int) {
			defer wg.Done()
			curNodeScore := sched.ScoreNodePlugins[extIndex](nodes, podInfo)

			mu.Lock()
			defer mu.Unlock()
			for i := range curNodeScore {
				nodeName := curNodeScore[i].Name
				score := curNodeScore[i].Score
				if allNodeScores[nodeName] == nil {
					allNodeScores[nodeName] = &NodeScore{
						Name:  nodeName,
						Score: 0,
					}
				}
				allNodeScores[nodeName].Score += score
				if allNodeScores[nodeName].Score > maxScore {
					maxScore = allNodeScores[nodeName].Score
				}
			}
		}(i)
	}

	wg.Wait()
	return getNodeScoresByScore(allNodeScores, maxScore)
}

func getNodeScoresByScore(nodeScore map[string]*NodeScore, score int64) []*NodeScore {
	var ns []*NodeScore
	for _, v := range nodeScore {
		if v.Score == score {
			ns = append(ns, v)
		}
	}
	return ns
}

// selectHost 从节点中选择一个节点
func (sched *Scheduler) selectHost(scores []*NodeScore) (*NodeScore, error) {
	if len(scores) == 0 {
		return nil, nil
	}

	return scores[0], nil
}

// bind 绑定pod到指定节点
func (sched *Scheduler) bind(podInfo *PodInfo, nodeName string) error {
	binding := &v1.Binding{
		ObjectMeta: metav1.ObjectMeta{Namespace: podInfo.Pod.Namespace, Name: podInfo.Pod.Name, UID: podInfo.Pod.UID},
		Target:     v1.ObjectReference{Kind: "Node", Name: nodeName},
	}
	err := kube.DefaultKubeClient().CoreV1().Pods(podInfo.Pod.Namespace).Bind(context.TODO(), binding, metav1.CreateOptions{})
	if err != nil {
		glog.Errorf("Error while binding podInfo %s: %v", podInfo.Pod.Name, err)
		return err
	}

	sched.AssumeCache.AddPod(nodeName, podInfo)
	return nil
}
