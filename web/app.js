(function () {
    "use strict";

    // 每个仪表盘的状态:当前显示值与目标值,用于平滑过渡。
    var gauges = {};

    function initGauges() {
        var canvases = document.querySelectorAll("canvas.gauge");
        canvases.forEach(function (cv) {
            var metric = cv.getAttribute("data-metric");
            gauges[metric] = {
                canvas: cv,
                ctx: cv.getContext("2d"),
                current: 0,
                target: 0,
            };
        });
        requestAnimationFrame(animate);
    }

    function colorFor(value) {
        if (value < 60) return "#3fd97f";
        if (value < 85) return "#ffb454";
        return "#ff5f56";
    }

    // 根据元素实际显示尺寸与设备像素比同步画布分辨率,保证清晰不模糊。
    function syncResolution(g) {
        var rect = g.canvas.getBoundingClientRect();
        var dpr = window.devicePixelRatio || 1;
        var w = Math.max(1, Math.round(rect.width * dpr));
        var h = Math.max(1, Math.round(rect.height * dpr));
        if (g.canvas.width !== w || g.canvas.height !== h) {
            g.canvas.width = w;
            g.canvas.height = h;
        }
    }

    function drawGauge(g) {
        syncResolution(g);

        var ctx = g.ctx;
        var size = Math.min(g.canvas.width, g.canvas.height);
        var cx = g.canvas.width / 2;
        var cy = g.canvas.height / 2;
        var lineWidth = size * 0.075;
        var radius = size / 2 - lineWidth;
        var value = g.current;

        ctx.clearRect(0, 0, g.canvas.width, g.canvas.height);

        var startAngle = 0.75 * Math.PI;
        var endAngle = 2.25 * Math.PI;
        var fullSweep = endAngle - startAngle;

        // 背景弧
        ctx.beginPath();
        ctx.arc(cx, cy, radius, startAngle, endAngle);
        ctx.lineWidth = lineWidth;
        ctx.lineCap = "round";
        ctx.strokeStyle = "#2a3340";
        ctx.stroke();

        // 进度弧
        var sweep = (value / 100) * fullSweep;
        ctx.beginPath();
        ctx.arc(cx, cy, radius, startAngle, startAngle + sweep);
        ctx.lineWidth = lineWidth;
        ctx.lineCap = "round";
        ctx.strokeStyle = colorFor(value);
        ctx.stroke();

        // 中心数值
        ctx.fillStyle = "#e6edf3";
        ctx.font = "600 " + Math.round(size * 0.2) + "px 'Segoe UI', sans-serif";
        ctx.textAlign = "center";
        ctx.textBaseline = "middle";
        ctx.fillText(Math.round(value) + "%", cx, cy);
    }

    function animate() {
        for (var key in gauges) {
            var g = gauges[key];
            var diff = g.target - g.current;
            if (Math.abs(diff) > 0.1) {
                g.current += diff * 0.15;
            } else {
                g.current = g.target;
            }
            drawGauge(g);
        }
        requestAnimationFrame(animate);
    }

    function setGauge(metric, value) {
        if (gauges[metric]) {
            gauges[metric].target = Math.max(0, Math.min(100, value));
        }
    }

    function formatBytes(bps) {
        if (bps < 1024) return bps.toFixed(0) + " B/s";
        if (bps < 1024 * 1024) return (bps / 1024).toFixed(1) + " KB/s";
        return (bps / (1024 * 1024)).toFixed(2) + " MB/s";
    }

    function formatGB(bytes) {
        return (bytes / (1024 * 1024 * 1024)).toFixed(1) + " GB";
    }

    function setStatus(online, text) {
        var dot = document.getElementById("statusDot");
        var label = document.getElementById("statusText");
        dot.className = "dot " + (online ? "online" : "offline");
        label.textContent = text;
    }

    function update(data) {
        setGauge("cpu", data.cpuPercent);
        setGauge("mem", data.mem.usedPercent);

        document.getElementById("cpuSub").textContent = "实时占用";
        document.getElementById("memSub").textContent =
            formatGB(data.mem.used) + " / " + formatGB(data.mem.total);

        if (data.gpuAvailable) {
            setGauge("gpu", data.gpuPercent);
            document.getElementById("gpuSub").textContent = "全部显卡综合";
        } else {
            setGauge("gpu", 0);
            document.getElementById("gpuSub").textContent = "无可用 GPU 计数器";
        }

        document.getElementById("netDown").textContent = formatBytes(data.net.downloadBps);
        document.getElementById("netUp").textContent = formatBytes(data.net.uploadBps);

        var d = new Date(data.timestamp);
        document.getElementById("updatedAt").textContent =
            "最后更新:" + d.toLocaleTimeString("zh-CN");
    }

    function connect() {
        var es = new EventSource("/events");

        es.onopen = function () {
            setStatus(true, "已连接");
        };

        es.onmessage = function (e) {
            try {
                update(JSON.parse(e.data));
                setStatus(true, "已连接");
            } catch (err) {
                /* 忽略解析错误 */
            }
        };

        es.onerror = function () {
            setStatus(false, "连接断开,重连中...");
            // EventSource 会自动重连,无需手动处理。
        };
    }

    function isFullscreen() {
        return !!(document.fullscreenElement || document.webkitFullscreenElement);
    }

    function requestFs() {
        var el = document.documentElement;
        var fn = el.requestFullscreen || el.webkitRequestFullscreen;
        if (fn) {
            try {
                var p = fn.call(el);
                if (p && typeof p.catch === "function") {
                    p.catch(function () { /* 浏览器可能因无用户手势而拒绝,忽略 */ });
                }
            } catch (err) { /* 忽略 */ }
        }
    }

    function exitFs() {
        var fn = document.exitFullscreen || document.webkitExitFullscreen;
        if (fn) {
            try { fn.call(document); } catch (err) { /* 忽略 */ }
        }
    }

    function toggleFullscreen() {
        if (isFullscreen()) {
            exitFs();
        } else {
            requestFs();
        }
    }

    function syncFsState() {
        document.body.classList.toggle("is-fullscreen", isFullscreen());
    }

    function initFullscreen() {
        var btn = document.getElementById("fullscreenBtn");
        if (btn) {
            btn.addEventListener("click", toggleFullscreen);
        }
        document.addEventListener("fullscreenchange", syncFsState);
        document.addEventListener("webkitfullscreenchange", syncFsState);

        // 默认尝试进入全屏(部分浏览器需用户手势,失败则在首次交互时再试)。
        requestFs();
        var onFirstInteract = function () {
            if (!isFullscreen()) {
                requestFs();
            }
            document.removeEventListener("click", onFirstInteract);
            document.removeEventListener("keydown", onFirstInteract);
            document.removeEventListener("touchstart", onFirstInteract);
        };
        document.addEventListener("click", onFirstInteract);
        document.addEventListener("keydown", onFirstInteract);
        document.addEventListener("touchstart", onFirstInteract);
    }

    document.addEventListener("DOMContentLoaded", function () {
        initGauges();
        initFullscreen();
        connect();
    });
})();
