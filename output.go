package main

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

// printStart 只打印根目录地址 + 总目录 + 退出提示
func printStart(units []*unit, rootDir string) {
	lan := lanIP()
	root := units[0]

	fmt.Println()
	fmt.Println("Servit 已启动")
	fmt.Println()

	fmt.Printf("  http://localhost:%d\n", root.port)
	fmt.Printf("  http://127.0.0.1:%d\n", root.port)
	if lan != "" {
		fmt.Printf("  http://%s:%d\n", lan, root.port)
	}

	fmt.Println()
	fmt.Printf("总目录: %s\n", rootDir)
	fmt.Println("按 Ctrl+C 优雅退出")
	fmt.Println()
}

// openBrowser 按平台打开默认浏览器
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		fmt.Println("自动打开浏览器失败，请手动访问:", url)
	}
}

// lanIP 返回本机局域网 IPv4 地址，找不到返回空
func lanIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil && !strings.HasPrefix(ip4.String(), "169.254.") {
				return ip4.String()
			}
		}
	}
	return ""
}
