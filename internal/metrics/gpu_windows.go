//go:build windows

package metrics

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	pdhDLL = windows.NewLazySystemDLL("pdh.dll")

	procPdhOpenQuery                = pdhDLL.NewProc("PdhOpenQueryW")
	procPdhAddEnglishCounter        = pdhDLL.NewProc("PdhAddEnglishCounterW")
	procPdhCollectQueryData         = pdhDLL.NewProc("PdhCollectQueryData")
	procPdhGetFormattedCounterArray = pdhDLL.NewProc("PdhGetFormattedCounterArrayW")
	procPdhCloseQuery               = pdhDLL.NewProc("PdhCloseQuery")
)

const (
	pdhFmtDouble        = 0x00000200
	pdhMoreData         = 0x800007D2
	pdhCStatusValidData = 0x00000000
	gpuCounterPath      = `\GPU Engine(*)\Utilization Percentage`
)

// pdhFmtCountervalueDouble 对应 PDH_FMT_COUNTERVALUE(double 形式)。
// 64 位下 CStatus(4 字节)后有 4 字节对齐填充,再接 8 字节 double。
type pdhFmtCountervalueDouble struct {
	CStatus     uint32
	_           uint32
	DoubleValue float64
}

// pdhFmtCountervalueItemDouble 对应 PDH_FMT_COUNTERVALUE_ITEM。
type pdhFmtCountervalueItemDouble struct {
	SzName *uint16
	Value  pdhFmtCountervalueDouble
}

// GPUMonitor 通过 Windows PDH 性能计数器读取 GPU 使用率,支持 NVIDIA/AMD/Intel。
type GPUMonitor struct {
	query   uintptr
	counter uintptr
	ok      bool
}

// NewGPUMonitor 创建并初始化 GPU 监控器。若系统无 GPU 计数器则 ok=false。
func NewGPUMonitor() *GPUMonitor {
	g := &GPUMonitor{}

	var query uintptr
	if r, _, _ := procPdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&query))); r != 0 {
		return g
	}

	counterPath, err := windows.UTF16PtrFromString(gpuCounterPath)
	if err != nil {
		procPdhCloseQuery.Call(query)
		return g
	}

	var counter uintptr
	if r, _, _ := procPdhAddEnglishCounter.Call(
		query,
		uintptr(unsafe.Pointer(counterPath)),
		0,
		uintptr(unsafe.Pointer(&counter)),
	); r != 0 {
		procPdhCloseQuery.Call(query)
		return g
	}

	// 利用率类计数器需要两次采样,先采集一次建立基线。
	procPdhCollectQueryData.Call(query)

	g.query = query
	g.counter = counter
	g.ok = true
	return g
}

// Usage 返回整机 GPU 使用率(百分比 0-100)与是否可用。
// 对所有 GPU Engine 实例求和并封顶 100%。
func (g *GPUMonitor) Usage() (float64, bool) {
	if !g.ok {
		return 0, false
	}

	if r, _, _ := procPdhCollectQueryData.Call(g.query); r != 0 {
		return 0, false
	}

	var bufferSize, itemCount uint32
	r, _, _ := procPdhGetFormattedCounterArray.Call(
		g.counter,
		uintptr(pdhFmtDouble),
		uintptr(unsafe.Pointer(&bufferSize)),
		uintptr(unsafe.Pointer(&itemCount)),
		0,
	)
	if r != pdhMoreData {
		// 暂无实例数据时返回 0(仍视为可用)。
		return 0, true
	}
	if itemCount == 0 || bufferSize == 0 {
		return 0, true
	}

	buf := make([]byte, bufferSize)
	r, _, _ = procPdhGetFormattedCounterArray.Call(
		g.counter,
		uintptr(pdhFmtDouble),
		uintptr(unsafe.Pointer(&bufferSize)),
		uintptr(unsafe.Pointer(&itemCount)),
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if r != 0 {
		return 0, false
	}

	items := unsafe.Slice((*pdhFmtCountervalueItemDouble)(unsafe.Pointer(&buf[0])), itemCount)
	var total float64
	for i := range items {
		if items[i].Value.CStatus == pdhCStatusValidData {
			total += items[i].Value.DoubleValue
		}
	}
	if total > 100 {
		total = 100
	}
	return total, true
}

// Close 释放 PDH 查询资源。
func (g *GPUMonitor) Close() {
	if g.ok {
		procPdhCloseQuery.Call(g.query)
		g.ok = false
	}
}
