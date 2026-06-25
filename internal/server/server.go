package server

import (
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"time"

	"windowsperformance/internal/metrics"
)

// Server 封装 HTTP 服务与指标采集器。
type Server struct {
	collector *metrics.Collector
	webFS     fs.FS
	interval  time.Duration
}

// New 创建服务。webFS 为内嵌的前端静态文件系统(根为 web 目录)。
func New(collector *metrics.Collector, webFS fs.FS, pushInterval time.Duration) *Server {
	if pushInterval <= 0 {
		pushInterval = time.Second
	}
	return &Server{
		collector: collector,
		webFS:     webFS,
		interval:  pushInterval,
	}
}

// Handler 返回配置好的 HTTP 路由。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(s.webFS)))
	mux.HandleFunc("/events", s.handleEvents)
	return mux
}

// handleEvents 通过 SSE 周期性推送最新指标。
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 连接建立后立即推送一帧,随后按间隔推送。
	if data, err := s.collector.LatestJSON(); err == nil {
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			data, err := s.collector.LatestJSON()
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// LocalIPs 返回本机内网 IPv4 地址列表(用于提示访问地址)。
func LocalIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			ips = append(ips, ip4.String())
		}
	}
	return ips
}

// PrimaryIP 返回本机用于内网访问的主 IPv4 地址。
// 优先通过默认路由(UDP 拨号探测)选出真实局域网网卡,
// 避免命中各类虚拟网卡(VMware/Hyper-V/WSL 等);失败时回退到私网地址优选。
func PrimaryIP() string {
	if conn, err := net.Dial("udp", "8.8.8.8:80"); err == nil {
		defer conn.Close()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			if ip4 := addr.IP.To4(); ip4 != nil && isPrivateIPv4(ip4) {
				return ip4.String()
			}
		}
	}
	return pickPrivateIP(LocalIPs())
}

// pickPrivateIP 从地址列表中优选一个内网地址,顺序:192.168 > 10 > 172.16-31 > 第一个。
func pickPrivateIP(ips []string) string {
	prefixes := []string{"192.168.", "10."}
	for _, p := range prefixes {
		for _, ip := range ips {
			if strings.HasPrefix(ip, p) {
				return ip
			}
		}
	}
	for _, ip := range ips {
		if ip4 := net.ParseIP(ip).To4(); ip4 != nil && isPrivateIPv4(ip4) {
			return ip
		}
	}
	if len(ips) > 0 {
		return ips[0]
	}
	return ""
}

// isPrivateIPv4 判断是否为 RFC1918 私网地址。
func isPrivateIPv4(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	switch {
	case ip4[0] == 10:
		return true
	case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
		return true
	case ip4[0] == 192 && ip4[1] == 168:
		return true
	default:
		return false
	}
}
