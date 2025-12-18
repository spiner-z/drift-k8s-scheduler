package plugin

import (
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	GPUPointsLabelKey       = "drift.io/gpu-points"
	GPUCountLabelKey        = "nvidia.com/gpu.count"
	GPUPointsPerCard  int64 = 1000
)

func gpuPointsUsedOnNode(ni *framework.NodeInfo) int64 {
	if ni == nil {
		return 0
	}
	var sum int64
	for _, pi := range ni.Pods {
		if pi == nil || pi.Pod == nil {
			continue
		}
		// 跳过已结束 Pod（一般 NodeInfo 里不会留着，但加了更稳）
		if pi.Pod.Status.Phase == v1.PodSucceeded || pi.Pod.Status.Phase == v1.PodFailed {
			continue
		}
		v := pi.Pod.Labels[GPUPointsLabelKey]
		if v == "" {
			continue
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			continue
		}
		sum += n
	}
	return sum
}

func gpuCapacityPointsFromLabel(ni *framework.NodeInfo) (int64, *framework.Status) {
	if ni == nil {
		return 0, framework.NewStatus(framework.Error, "nil nodeInfo")
	}
	node := ni.Node()
	if node == nil {
		return 0, framework.NewStatus(framework.Error, "node is nil in nodeInfo")
	}

	s := node.Labels[GPUCountLabelKey]
	if s == "" {
		// 没打这个标签，就认为 gpu_count=0
		return 0, nil
	}

	gpuCount, err := strconv.ParseInt(s, 10, 64)
	if err != nil || gpuCount < 0 {
		return 0, framework.NewStatus(
			framework.UnschedulableAndUnresolvable,
			"invalid node label nvidia.com/gpu.count (must be non-negative int)",
		)
	}
	return gpuCount * GPUPointsPerCard, nil
}

func gpuPointsFromPod(pod *v1.Pod) (int64, *framework.Status) {
	if pod == nil {
		return 0, framework.NewStatus(framework.Error, "nil pod")
	}
	val, ok := pod.Labels[GPUPointsLabelKey]
	if !ok || val == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, framework.NewStatus(framework.UnschedulableAndUnresolvable, "invalid gpu-points label (must be int64)")
	}
	if n < 0 || n > GPUPointsPerCard {
		return 0, framework.NewStatus(framework.UnschedulableAndUnresolvable, "gpu-points must be in [0,1000]")
	}
	return n, nil
}
