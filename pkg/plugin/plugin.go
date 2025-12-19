// plugin.go
package plugin

import (
	"context"
	"fmt"

	simontype "github.com/spiner-z/drift-k8s-scheduler/pkg/type"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework" // 调度框架接口定义
)

const PluginName = "DriftPlugin"

func New(ctx context.Context, cfg runtime.Object, h framework.Handle) (framework.Plugin, error) {
	_ = ctx
	_ = cfg
	t := initTypicalPods()
	return &DriftPlugin{handle: h, typicalPods: t}, nil
}

// 插件结构体定义
type DriftPlugin struct {
	handle      framework.Handle // 调度器可以通过 handle 访问集群状态等，暂不使用
	typicalPods *simontype.TargetPodList
}

// Name 方法返回插件名称
func (dp *DriftPlugin) Name() string {
	return "DriftPlugin"
}

/*
Filter:
- 如果 Pod 没有 gpu-points（或为0），直接 Success
- 否则：sum(node上所有pod的gpu-points) + 当前pod的gpu-points <= 1000 * gpu_count
- 其它原生 Filter（资源、亲和性、污点等）不在这里做，交给默认插件链
*/
// func (dp *DriftPlugin) Filter(ctx context.Context, state *framework.CycleState,
// 	pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
// 	need, st := gpuPointsFromPod(pod)
// 	if st != nil && !st.IsSuccess() {
// 		return st
// 	}
// 	if need == 0 {
// 		// Pod 不需要 GPU，直接通过
// 		return framework.NewStatus(framework.Success, "")
// 	}
// 	if need > GPUPointsPerCard {
// 		return framework.NewStatus(
// 			framework.Unschedulable,
// 			fmt.Sprintf("pod gpu-points %d exceeds per-card limit %d", need, GPUPointsPerCard),
// 		)
// 	}
// 	capPoints, st := gpuCapacityPointsFromLabel(nodeInfo)
// 	if st != nil && !st.IsSuccess() {
// 		return st
// 	}
// 	if capPoints <= 0 {
// 		return framework.NewStatus(framework.Unschedulable, "node has 0 allocatable GPUs for gpu-points sharing")
// 	}
// 	usedPoints := gpuPointsUsedOnNode(nodeInfo)
// 	if usedPoints+need > capPoints {
// 		return framework.NewStatus(
// 			framework.Unschedulable,
// 			fmt.Sprintf("gpu-points exceeded: used=%d need=%d cap=%d", usedPoints, need, capPoints),
// 		)
// 	}
// 	return framework.NewStatus(framework.Success, "")
// }
func (dp *DriftPlugin) Filter(ctx context.Context, state *framework.CycleState,
	pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	need, st := gpuPointsFromPod(pod)
	if st != nil && !st.IsSuccess() {
		return st
	}
	if need == 0 {
		// Pod 不需要 GPU，直接通过
		return framework.NewStatus(framework.Success, "")
	}
	if need > GPUPointsPerCard {
		return framework.NewStatus(
			framework.Unschedulable,
			fmt.Sprintf("pod gpu-points %d exceeds per-card limit %d", need, GPUPointsPerCard),
		)
	}
	nodeRes := GetNodeResourceViaNodeInfo(nodeInfo)
	podRes := GetPodResource(pod)
	if !IsNodeAccessibleToPod(nodeRes, podRes) {
		return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("Node (%s) %s does not match GPU type request of pod %s\n", nodeInfo.Node().Name, nodeRes.Repr(), podRes.Repr()))
	}
	return framework.NewStatus(framework.Success, "")
}

// func (dp *DriftPlugin) Score(ctx context.Context, state *framework.CycleState,
// 	pod *v1.Pod, nodeName string) (int64, *framework.Status) {

// 	// 通过 handle 或框架获取 nodeInfo
// 	nodeInfo, err := dp.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
// 	if err != nil {
// 		// 获取节点信息出错，返回错误状态
// 		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("获取节点信息失败: %v", err))
// 	}

// 	allocable := nodeInfo.Allocatable // 可分配资源
// 	requested := nodeInfo.Requested   // 已请求资源

// 	// 避免除零错误
// 	if allocable.MilliCPU == 0 || allocable.Memory == 0 {
// 		return 0, framework.NewStatus(framework.Error, "节点资源数据异常")
// 	}

// 	// 计算CPU和内存剩余百分比（0~1之间）
// 	freeCPUFrac := float64(allocable.MilliCPU-requested.MilliCPU) / float64(allocable.MilliCPU)
// 	freeMemFrac := float64(allocable.Memory-requested.Memory) / float64(allocable.Memory)

// 	if freeCPUFrac < 0 {
// 		freeCPUFrac = 0
// 	}
// 	if freeMemFrac < 0 {
// 		freeMemFrac = 0
// 	}

// 	// 按权重计算综合得分（CPU权重0.8，内存权重0.2）
// 	rate := freeCPUFrac*0.8 + freeMemFrac*0.2

// 	needGPU, st := gpuPointsFromPod(pod)
// 	if st != nil {
// 		return 0, st
// 	}

// 	finalRate := rate
// 	if needGPU > 0 {
// 		// Pod 需要 GPU，考虑 GPU 资源
// 		capPoints, st := gpuCapacityPointsFromLabel(nodeInfo)
// 		if st != nil {
// 			return 0, st
// 		}
// 		usedPoints := gpuPointsUsedOnNode(nodeInfo)
// 		gpuRate := float64(capPoints-usedPoints) / float64(capPoints)
// 		if gpuRate < 0 {
// 			gpuRate = 0
// 		}
// 		// 综合 CPU、内存和 GPU 得分（GPU 权重0.4，CPU+内存权重0.6）
// 		finalRate = (rate*0.6 + gpuRate*0.4)
// 	}

// 	// 转换为调度器要求的整数分值 (0~100)
// 	score := int64(finalRate * 100)
// 	if score < 0 {
// 		score = 0
// 	}
// 	if score > 100 {
// 		score = 100
// 	}
// 	return score, framework.NewStatus(framework.Success, "")
// }

func (dp *DriftPlugin) Score(ctx context.Context, state *framework.CycleState,
	pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	var nodeRes simontype.NodeResource
	var podRes simontype.PodResource
	// 通过 handle 或框架获取 nodeInfo
	nodeInfo, err := dp.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		// 获取节点信息出错，返回错误状态
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("获取节点信息失败: %v", err))
	}
	nodeRes = GetNodeResourceViaNodeInfo(nodeInfo)
	podRes = GetPodResource(pod)
	if !IsNodeAccessibleToPod(nodeRes, podRes) {
		return framework.MinNodeScore, framework.NewStatus(framework.Error, fmt.Sprintf("Node (%s) %s does not match GPU type request of pod %s\n", nodeName, nodeRes.Repr(), podRes.Repr()))
	}
	score := calculateGpuShareFragExtendScore(nodeRes, podRes, dp.typicalPods)
	return score, framework.NewStatus(framework.Success)
}

// ScoreExtensions 可选接口，用于提供NormalizeScore等能力。这里不需要额外处理，返回 nil。
func (dp *DriftPlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

func calculateGpuShareFragExtendScore(nodeRes simontype.NodeResource, podRes simontype.PodResource, typicalPods *simontype.TargetPodList) (score int64) {
	nodeGpuShareFragScore := NodeGpuShareFragAmountScore(nodeRes, *typicalPods)
	if podRes.GpuNumber == 1 && podRes.MilliGpu < GPUPointsPerCard { // request partial GPU
		score = 0
		for i := 0; i < len(nodeRes.MilliGpuLeftList); i++ {
			if nodeRes.MilliGpuLeftList[i] >= podRes.MilliGpu {
				newNodeRes := nodeRes.Copy()
				newNodeRes.MilliCpuLeft -= podRes.MilliCpu
				newNodeRes.MilliGpuLeftList[i] -= podRes.MilliGpu
				newNodeGpuShareFragScore := NodeGpuShareFragAmountScore(newNodeRes, *typicalPods)
				fragScore := int64(sigmoid((nodeGpuShareFragScore-newNodeGpuShareFragScore)/1000) * float64(framework.MaxNodeScore))
				if fragScore > score {
					score = fragScore
				}
			}
		}
		return score
	} else {
		newNodeRes, _ := nodeRes.Sub(podRes)
		newNodeGpuShareFragScore := NodeGpuShareFragAmountScore(newNodeRes, *typicalPods)
		return int64(sigmoid((nodeGpuShareFragScore-newNodeGpuShareFragScore)/1000) * float64(framework.MaxNodeScore))
	}
}
