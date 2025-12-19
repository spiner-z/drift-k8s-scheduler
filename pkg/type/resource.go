package simontype

import (
	"fmt"
	"sort"
)

const (
	DefaultTypicalPodPopularityThreshold = 60 // 60%
	DefaultTypicalPodIncreaseStep        = 10
)

type TargetPod struct {
	TargetPodResource PodResource
	Percentage        float64 // range: 0.0 - 1.0 (100%)
}

type TargetPodList []TargetPod

type SkylinePodList []PodResource

func (p TargetPodList) Len() int { return len(p) }
func (p TargetPodList) Less(i, j int) bool {
	if p[i].Percentage != p[j].Percentage {
		return p[i].Percentage < p[j].Percentage
	} else { // to stabilize the order if two TargetPod has the same frequency
		return p[i].TargetPodResource.Less(p[j].TargetPodResource)
	}
}
func (tpr PodResource) Less(other PodResource) bool {
	if tpr.MilliCpu != other.MilliCpu {
		return tpr.MilliCpu < other.MilliCpu
	} else if tpr.MilliGpu != other.MilliGpu {
		return tpr.MilliGpu < other.MilliGpu
	} else if tpr.GpuNumber != other.GpuNumber {
		return tpr.GpuNumber < other.GpuNumber
	} else {
		return tpr.GpuType < other.GpuType
	}
}
func (p TargetPodList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

type PodResource struct { // typical pod, without name and namespace.
	MilliCpu  int64
	MilliGpu  int64 // Milli GPU request per GPU, 0-1000
	GpuNumber int
	GpuType   string
	//Memory	  int64
}

func (tpr PodResource) Repr() string {
	outStr := "<"
	outStr += fmt.Sprintf("CPU: %6.2f", float64(tpr.MilliCpu)/1000)
	outStr += fmt.Sprintf(", GPU: %d", tpr.GpuNumber)
	outStr += fmt.Sprintf(" x {%-4d}m", tpr.MilliGpu)
	outStr += fmt.Sprintf(" (%s)", tpr.GpuType)
	outStr += ">"
	return outStr
}

type NodeResource struct {
	NodeName         string
	MilliCpuLeft     int64
	MilliCpuCapacity int64
	MilliGpuLeftList []int64 // Do NOT sort it directly, using SortedMilliGpuLeftIndexList instead. Its order matters; the index is the GPU device index.
	GpuNumber        int
	GpuType          string
	GpuAffinity      map[string]int
	// MemoryLeft       int64
	// MemoryCapacity   int64
}

func (tnr NodeResource) Copy() NodeResource {
	milliGpuLeftList := make([]int64, len(tnr.MilliGpuLeftList))
	for i := 0; i < len(tnr.MilliGpuLeftList); i++ {
		milliGpuLeftList[i] = tnr.MilliGpuLeftList[i]
	}

	return NodeResource{
		NodeName:         tnr.NodeName,
		MilliCpuLeft:     tnr.MilliCpuLeft,
		MilliCpuCapacity: tnr.MilliCpuCapacity,
		MilliGpuLeftList: milliGpuLeftList,
		GpuNumber:        tnr.GpuNumber,
		GpuType:          tnr.GpuType,
		GpuAffinity:      tnr.GpuAffinity,
	}
}

func (tnr NodeResource) Sub(tpr PodResource) (NodeResource, error) {
	out := tnr.Copy()
	if out.MilliCpuLeft < tpr.MilliCpu || out.GpuNumber < tpr.GpuNumber {
		return out, fmt.Errorf("node: %s failed to accommodate pod: %s", tnr.Repr(), tpr.Repr())
	}
	out.MilliCpuLeft -= tpr.MilliCpu

	gpuRequest := tpr.GpuNumber
	if gpuRequest == 0 {
		return out, nil
	}

	// Sort NodeRes's GpuLeft in ascending, then Pack it (Subtract from the least sufficient one).
	sortedIndex := out.SortedMilliGpuLeftIndexList(true)

	for i := 0; i < len(sortedIndex); i++ {
		if tpr.MilliGpu <= out.MilliGpuLeftList[sortedIndex[i]] {
			gpuRequest -= 1
			out.MilliGpuLeftList[sortedIndex[i]] -= tpr.MilliGpu
			if gpuRequest <= 0 {
				//fmt.Printf("[DEBUG] [DONE] out.MilliGpuLeftList: %v\n", out.MilliGpuLeftList)
				return out, nil
			}
		}
	}
	return out, fmt.Errorf("node: %s failed to accommodate pod: %s (%d GPU requests left)", tnr.Repr(), tpr.Repr(), gpuRequest)
}

func (tnr NodeResource) SortedMilliGpuLeftIndexList(ascending bool) []int {
	indexList := make([]int, len(tnr.MilliGpuLeftList)) // len == cap == len(tnr.MilliGpuLeftList)
	for i, _ := range tnr.MilliGpuLeftList {
		indexList[i] = i
	}

	if ascending {
		// smallest one (minimum GPU Milli left) first
		sort.SliceStable(indexList, func(i, j int) bool {
			return tnr.MilliGpuLeftList[indexList[i]] < tnr.MilliGpuLeftList[indexList[j]]
		})
	} else {
		// largest one (maximum GPU Milli left) first
		sort.SliceStable(indexList, func(i, j int) bool {
			return tnr.MilliGpuLeftList[indexList[i]] > tnr.MilliGpuLeftList[indexList[j]]
		})
	}
	return indexList
}

func (tnr NodeResource) Repr() string {
	outStr := fmt.Sprintf("%s<", tnr.NodeName)
	outStr += fmt.Sprintf("CPU: %6.2f/%6.2f", float64(tnr.MilliCpuLeft)/1000, float64(tnr.MilliCpuCapacity)/1000)
	outStr += fmt.Sprintf(", GPU (%s): %d", tnr.GpuType, tnr.GpuNumber)
	if tnr.GpuNumber > 0 {
		outStr += fmt.Sprintf(" x %dm, Left:", 1000)
		for _, gML := range tnr.MilliGpuLeftList {
			outStr += fmt.Sprintf(" %dm", gML)
		}
	}
	outStr += ">"
	return outStr
}
