//go:build darwin

package metrics

import (
	"os/exec"
	"regexp"
	"strconv"
)

// GPUMonitor 在 macOS 上通过 ioreg 读取 GPU 利用率,兼容 Intel 与 Apple Silicon,无需 sudo。
type GPUMonitor struct {
	re *regexp.Regexp
}

// NewGPUMonitor 创建 macOS GPU 监控器。
func NewGPUMonitor() *GPUMonitor {
	return &GPUMonitor{
		// 兼容不同芯片/驱动的字段命名。
		re: regexp.MustCompile(`"(?:Device Utilization %|GPU Activity\(%\)|GPU Core Utilization)"\s*=\s*(\d+(?:\.\d+)?)`),
	}
}

// Usage 返回 GPU 使用率(百分比 0-100)与是否可用。多 GPU 时取最大值。
func (g *GPUMonitor) Usage() (float64, bool) {
	out, err := exec.Command("ioreg", "-r", "-d", "1", "-w", "0", "-c", "IOAccelerator").Output()
	if err != nil {
		return 0, false
	}

	matches := g.re.FindAllStringSubmatch(string(out), -1)
	if len(matches) == 0 {
		return 0, false
	}

	var maxUsage float64
	found := false
	for _, m := range matches {
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			continue
		}
		found = true
		if v > maxUsage {
			maxUsage = v
		}
	}
	if !found {
		return 0, false
	}
	if maxUsage > 100 {
		maxUsage = 100
	}
	return maxUsage, true
}

// Close 在 macOS 平台为空操作。
func (g *GPUMonitor) Close() {}
