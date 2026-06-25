package metrics

import (
	"encoding/json"
	"sync"
	"time"
)

// Snapshot 表示某一时刻的完整性能指标快照。
type Snapshot struct {
	Timestamp   int64    `json:"timestamp"`   // Unix 毫秒时间戳
	CPUPercent  float64  `json:"cpuPercent"`  // CPU 使用率(0-100)
	Mem         MemInfo  `json:"mem"`         // 内存信息
	Net         NetSpeed `json:"net"`         // 网络速率
	GPUPercent  float64  `json:"gpuPercent"`  // GPU 使用率(0-100)
	GPUAvailable bool    `json:"gpuAvailable"` // GPU 数据是否可用
}

// Collector 在后台定时采集指标,并保存最新快照供并发读取。
type Collector struct {
	interval time.Duration
	gpu      *GPUMonitor

	mu      sync.RWMutex
	latest  Snapshot
	stopCh  chan struct{}
	started bool
}

// NewCollector 创建一个采集器,interval 为采集间隔。
func NewCollector(interval time.Duration) *Collector {
	if interval <= 0 {
		interval = time.Second
	}
	return &Collector{
		interval: interval,
		gpu:      NewGPUMonitor(),
		stopCh:   make(chan struct{}),
	}
}

// Start 启动后台采集 goroutine。
func (c *Collector) Start() {
	if c.started {
		return
	}
	c.started = true

	// 立即采集一次,建立 CPU/网速基线并尽快有数据。
	c.collectOnce()

	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.collectOnce()
			case <-c.stopCh:
				return
			}
		}
	}()
}

// Stop 停止采集并释放资源。
func (c *Collector) Stop() {
	if !c.started {
		return
	}
	close(c.stopCh)
	c.gpu.Close()
	c.started = false
}

func (c *Collector) collectOnce() {
	gpuPercent, gpuOK := c.gpu.Usage()
	snap := Snapshot{
		Timestamp:    time.Now().UnixMilli(),
		CPUPercent:   CPUUsage(),
		Mem:          Memory(),
		Net:          Network(),
		GPUPercent:   gpuPercent,
		GPUAvailable: gpuOK,
	}

	c.mu.Lock()
	c.latest = snap
	c.mu.Unlock()
}

// Latest 返回最新的指标快照。
func (c *Collector) Latest() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latest
}

// LatestJSON 返回最新快照的 JSON 编码。
func (c *Collector) LatestJSON() ([]byte, error) {
	return json.Marshal(c.Latest())
}
