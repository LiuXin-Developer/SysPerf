package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/mdp/qrterminal/v3"

	"windowsperformance/internal/metrics"
	"windowsperformance/internal/server"
)

//go:embed web
var embeddedFiles embed.FS

func main() {
	port := flag.Int("port", 8080, "Web 服务监听端口")
	interval := flag.Int("interval", 1000, "采集与推送间隔(毫秒)")
	flag.Parse()

	webFS, err := fs.Sub(embeddedFiles, "web")
	if err != nil {
		fmt.Println("内嵌前端资源加载失败:", err)
		os.Exit(1)
	}

	pollInterval := time.Duration(*interval) * time.Millisecond

	collector := metrics.NewCollector(pollInterval)
	collector.Start()
	defer collector.Stop()

	srv := server.New(collector, webFS, pollInterval)

	addr := ":" + strconv.Itoa(*port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: srv.Handler(),
	}

	printBanner(*port)

	// 优雅退出:监听 Ctrl+C。
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)
		<-stop
		fmt.Println("\n正在关闭服务...")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)
	}()

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Println("服务启动失败:", err)
		os.Exit(1)
	}
}

func printBanner(port int) {
	line := "============================================================"
	fmt.Println(line)
	fmt.Println("  系统性能监控  已启动")
	fmt.Println(line)
	fmt.Printf("  本机访问:   http://localhost:%d\n", port)

	lanIP := server.PrimaryIP()
	var lanURL string
	if lanIP == "" {
		fmt.Println("  内网访问:   未检测到内网 IP")
	} else {
		lanURL = fmt.Sprintf("http://%s:%d", lanIP, port)
		fmt.Printf("  内网访问:   %s\n", lanURL)
	}
	fmt.Println(line)

	if lanURL != "" {
		fmt.Println("  扫码访问(手机与本机需在同一局域网):")
		printQRCode(lanURL)
	}

	fmt.Println("  内网内其它设备可用上述「内网访问」地址打开仪表盘")
	fmt.Println("  按 Ctrl+C 退出")
	fmt.Println(line)
}

func printQRCode(url string) {
	// 使用半块字符渲染,体积更小且更易被手机相机识别。
	qrterminal.GenerateHalfBlock(url, qrterminal.L, os.Stdout)
}
