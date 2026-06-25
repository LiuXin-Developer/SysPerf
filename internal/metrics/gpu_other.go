//go:build !windows && !darwin

package metrics

// GPUMonitor 在 Windows / macOS 以外的平台为空实现(如 Linux)。
type GPUMonitor struct{}

// NewGPUMonitor 返回不可用的监控器。
func NewGPUMonitor() *GPUMonitor { return &GPUMonitor{} }

// Usage 始终返回不可用。
func (g *GPUMonitor) Usage() (float64, bool) { return 0, false }

// Close 为空操作。
func (g *GPUMonitor) Close() {}
