// plugin.go
package plugin

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework" // 调度框架接口定义
)

const PluginName = "DriftPlugin"

func New(ctx context.Context, cfg runtime.Object, h framework.Handle) (framework.Plugin, error) {
	_ = ctx
	_ = cfg
	return &DriftPlugin{handle: h}, nil
}

// 插件结构体定义
type DriftPlugin struct {
	handle framework.Handle // 调度器可以通过 handle 访问集群状态等，暂不使用
}

// Name 方法返回插件名称
func (dp *DriftPlugin) Name() string {
	return "DriftPlugin"
}

// Filter 方法：节点可用性过滤逻辑
func (dp *DriftPlugin) Filter(ctx context.Context, state *framework.CycleState,
	pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {

	// 获取节点总资源和已分配资源
	allocable := nodeInfo.Allocatable // 可分配资源
	requested := nodeInfo.Requested   // 已请求资源

	// 计算节点剩余资源
	freeCPU := allocable.MilliCPU - requested.MilliCPU // 剩余 CPU (毫核)
	freeMem := allocable.Memory - requested.Memory     // 剩余内存 (字节)

	// 计算待调度Pod所需 CPU 和内存总请求量
	var podCPU int64 = 0
	var podMem int64 = 0
	for _, c := range pod.Spec.Containers {
		// 累加每个容器的 Requests 资源
		cpuQty := c.Resources.Requests.Cpu()    // v1.ResourceList中CPU数量
		memQty := c.Resources.Requests.Memory() // 内存
		podCPU += cpuQty.MilliValue()           // CPU以m单位
		podMem += memQty.Value()                // 内存字节值
	}

	// 判断节点是否有足够余量容纳该Pod
	if freeCPU < podCPU || freeMem < podMem {
		// 节点资源不足，返回 Unschedulable 状态和原因
		return framework.NewStatus(framework.Unschedulable, "节点资源不足，无法调度该Pod")
	}

	// 资源充足，节点通过筛选
	return framework.NewStatus(framework.Success, "")
}

// Score 方法：节点评分逻辑（0-100分，分值越高表示越适合）
func (dp *DriftPlugin) Score(ctx context.Context, state *framework.CycleState,
	pod *v1.Pod, nodeName string) (int64, *framework.Status) {

	// 通过 handle 或框架获取 nodeInfo
	nodeInfo, err := dp.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		// 获取节点信息出错，返回错误状态
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("获取节点信息失败: %v", err))
	}

	// alloc := nodeInfo.AllocatableResource()
	// used := nodeInfo.RequestedResource()
	allocable := nodeInfo.Allocatable // 可分配资源
	requested := nodeInfo.Requested   // 已请求资源

	// 避免除零错误
	if allocable.MilliCPU == 0 || allocable.Memory == 0 {
		return 0, framework.NewStatus(framework.Error, "节点资源数据异常")
	}

	// 计算CPU和内存剩余百分比（0~1之间）
	freeCPUFrac := float64(allocable.MilliCPU-requested.MilliCPU) / float64(allocable.MilliCPU)
	freeMemFrac := float64(allocable.Memory-requested.Memory) / float64(allocable.Memory)

	if freeCPUFrac < 0 {
		freeCPUFrac = 0
	}
	if freeMemFrac < 0 {
		freeMemFrac = 0
	}

	// 按权重计算综合得分（CPU权重0.8，内存权重0.2）
	totalScore := freeCPUFrac*0.8 + freeMemFrac*0.2

	// 转换为调度器要求的整数分值 (0~100)
	score := int64(totalScore * 100)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score, framework.NewStatus(framework.Success, "")
}

// ScoreExtensions 可选接口，用于提供NormalizeScore等能力。这里不需要额外处理，返回 nil。
func (dp *DriftPlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}
