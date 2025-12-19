package simontype

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
