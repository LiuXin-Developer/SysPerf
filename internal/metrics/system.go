package metrics

import (
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// netState 用于在两次采样之间保存网络累计字节数,从而计算速率。
type netState struct {
	lastBytesSent uint64
	lastBytesRecv uint64
	lastTime      time.Time
	initialized   bool
}

var globalNetState netState

// CPUUsage 返回 CPU 总使用率(百分比 0-100)。
// 传入 0 表示返回自上次调用以来的使用率。
func CPUUsage() float64 {
	percents, err := cpu.Percent(0, false)
	if err != nil || len(percents) == 0 {
		return 0
	}
	return percents[0]
}

// MemInfo 返回内存使用情况。
type MemInfo struct {
	Total       uint64  `json:"total"`       // 总内存(字节)
	Used        uint64  `json:"used"`        // 已用内存(字节)
	UsedPercent float64 `json:"usedPercent"` // 已用百分比
}

// Memory 返回当前内存使用情况。
func Memory() MemInfo {
	v, err := mem.VirtualMemory()
	if err != nil {
		return MemInfo{}
	}
	return MemInfo{
		Total:       v.Total,
		Used:        v.Used,
		UsedPercent: v.UsedPercent,
	}
}

// NetSpeed 返回网络上行/下行速率。
type NetSpeed struct {
	UploadBps   float64 `json:"uploadBps"`   // 上行速率(字节/秒)
	DownloadBps float64 `json:"downloadBps"` // 下行速率(字节/秒)
}

// Network 计算自上次调用以来的网络收发速率。
// 第一次调用会初始化基线并返回 0。
func Network() NetSpeed {
	counters, err := net.IOCounters(false)
	if err != nil || len(counters) == 0 {
		return NetSpeed{}
	}
	now := time.Now()
	sent := counters[0].BytesSent
	recv := counters[0].BytesRecv

	if !globalNetState.initialized {
		globalNetState = netState{
			lastBytesSent: sent,
			lastBytesRecv: recv,
			lastTime:      now,
			initialized:   true,
		}
		return NetSpeed{}
	}

	elapsed := now.Sub(globalNetState.lastTime).Seconds()
	if elapsed <= 0 {
		return NetSpeed{}
	}

	var upDelta, downDelta uint64
	if sent >= globalNetState.lastBytesSent {
		upDelta = sent - globalNetState.lastBytesSent
	}
	if recv >= globalNetState.lastBytesRecv {
		downDelta = recv - globalNetState.lastBytesRecv
	}

	globalNetState.lastBytesSent = sent
	globalNetState.lastBytesRecv = recv
	globalNetState.lastTime = now

	return NetSpeed{
		UploadBps:   float64(upDelta) / elapsed,
		DownloadBps: float64(downDelta) / elapsed,
	}
}
