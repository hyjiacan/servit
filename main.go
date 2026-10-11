package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const usageText = `Servit —— 双击即用的本地多端口静态服务器

用法:
  servit [选项]

选项:
  -port int              根目录端口（默认 8080，被占用自动递增）
                         未指定端口的子目录从其后依次分配
  -dir string            总目录（默认 exe目录/www）
                         其下每个一级子目录各起一个独立服务
  -no-open               启动后不自动打开浏览器
  -spa                   每个单元开启 SPA 路由回退
  -cors                  开启 CORS 响应头（默认开启）
  -shutdown-timeout int  优雅退出等待秒数（默认 5，超时强制关闭）
  -h, --help             显示本帮助

示例:
  servit                      双击或直接运行，全默认
  servit -port 9000           根端口 9000，未指定端口的子目录从 9001 起
  servit -dir ./projects      指定总目录
  servit -no-open             只起服务，不开浏览器

说明:
  首次运行会在 exe 同目录生成 www/ 并释放内置示例与帮助。
  www/ 已存在时不会覆盖。
  根目录提供所有一级子目录的索引，每个子目录独立端口访问。
  可在子目录下放 servit.conf 自定义标题、描述、端口：
      title=我的项目
      desc=这是一个演示
      port=9001
  按 Ctrl+C 优雅退出。
`

func main() {
	for _, a := range os.Args[1:] {
		if a == "-h" || a == "--help" {
			fmt.Print(usageText)
			return
		}
	}

	var (
		port            = flag.Int("port", 8080, "根目录端口")
		dir             = flag.String("dir", "", "总目录")
		noOpen          = flag.Bool("no-open", false, "不自动打开浏览器")
		spa             = flag.Bool("spa", false, "SPA 路由回退")
		cors            = flag.Bool("cors", true, "开启 CORS 响应头")
		shutdownTimeout = flag.Int("shutdown-timeout", 5, "优雅退出等待秒数")
	)
	flag.Usage = func() { fmt.Print(usageText) }
	flag.Parse()

	// 定位总目录
	absRoot, err := resolveRootDir(*dir)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// 释放内置内容
	if !dirExists(absRoot) {
		fmt.Printf("未发现总目录，正在释放内置示例到: %s\n", absRoot)
		if err := releaseSeed(absRoot); err != nil {
			log.Fatalf("释放内置内容失败: %v", err)
		}
		fmt.Println("释放完成。")
	}

	// 扫描并构建单元
	units, err := buildUnits(absRoot)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// 分配端口
	if err := assignPorts(units, *port); err != nil {
		log.Fatalf("%v", err)
	}

	// 填充子目录端口映射，供索引页使用
	fillSubdirPorts(units, absRoot)

	// 构建 handler
	buildHandlers(units, absRoot, *spa, *cors)

	// 启动所有服务
	for _, u := range units {
		go func(u *unit) {
			if err := u.server.Serve(u.listener); err != nil && err.Error() != "http: Server closed" {
				fmt.Printf("[%d] 服务异常退出: %v\n", u.port, err)
			}
		}(u)
	}

	printStart(units, absRoot)

	if !*noOpen {
		go func() {
			time.Sleep(300 * time.Millisecond)
			openBrowser(fmt.Sprintf("http://localhost:%d", units[0].port))
		}()
	}

	// 信号监听
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh
	fmt.Println("\n正在优雅退出...")

	go func() {
		<-sigCh
		fmt.Println("再次收到信号，强制退出。")
		os.Exit(1)
	}()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(*shutdownTimeout)*time.Second,
	)
	defer cancel()

	var wg sync.WaitGroup
	for _, u := range units {
		wg.Add(1)
		go func(u *unit) {
			defer wg.Done()
			if err := u.server.Shutdown(ctx); err != nil {
				u.server.Close()
				fmt.Printf("[%d] 等待超时，已强制停止\n", u.port)
				return
			}
			fmt.Printf("[%d] 已停止\n", u.port)
		}(u)
	}
	wg.Wait()

	fmt.Println("已退出，感谢使用 Servit。")
}

// resolveRootDir 确定总目录的绝对路径
func resolveRootDir(dirFlag string) (string, error) {
	root := dirFlag
	if root == "" {
		exePath, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("无法定位程序路径: %w", err)
		}
		if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
			exePath = resolved
		}
		root = filepath.Join(filepath.Dir(exePath), "www")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("无法解析总目录: %w", err)
	}
	return abs, nil
}
