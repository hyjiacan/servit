package main

import (
	"fmt"
	"net"
	"net/http"
)

type unit struct {
	name     string
	dir      string
	port     int
	title    string
	desc     string
	custom   bool
	listener net.Listener
	server   *http.Server
}

// 全局端口映射：子目录名 → 端口，供根目录索引页查询
var subdirPorts = map[string]int{}

// buildUnits 扫描总目录，构建根单元 + 每个子目录单元
func buildUnits(absRoot string) ([]*unit, error) {
	subdirs, err := scanSubdirs(absRoot)
	if err != nil {
		return nil, fmt.Errorf("扫描子目录失败: %w", err)
	}

	// 根单元
	rootMeta := readMeta(absRoot)
	rootUnit := &unit{name: "www (根)", dir: absRoot, title: "根目录"}
	if rootMeta.title != "" {
		rootUnit.title = rootMeta.title
	}
	if rootMeta.desc != "" {
		rootUnit.desc = rootMeta.desc
	}
	if rootMeta.port != 0 {
		rootUnit.port = rootMeta.port
		rootUnit.custom = true
	}
	units := []*unit{rootUnit}

	// 子目录单元
	for _, d := range subdirs {
		name := baseName(d)
		m := readMeta(d)
		title := m.title
		if title == "" {
			title = name
		}
		u := &unit{
			name:  name,
			dir:   d,
			title: title,
			desc:  m.desc,
		}
		if m.port != 0 {
			u.port = m.port
			u.custom = true
		}
		units = append(units, u)
	}

	return units, nil
}

// assignPorts 为所有单元分配端口
func assignPorts(units []*unit, startPort int) error {
	used := map[int]bool{}

	// 第一轮：处理自定义端口
	for _, u := range units {
		if !u.custom {
			continue
		}
		if used[u.port] {
			return fmt.Errorf("端口 %d 被多个单元重复指定", u.port)
		}
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", u.port))
		if err != nil {
			return fmt.Errorf("单元 %s 指定的端口 %d 无法使用: %w", u.name, u.port, err)
		}
		u.listener = ln
		used[u.port] = true
	}

	// 第二轮：其余单元依次分配
	nextPort := startPort
	for _, u := range units {
		if u.custom {
			continue
		}
		for {
			if used[nextPort] {
				nextPort++
				continue
			}
			ln, err := net.Listen("tcp", fmt.Sprintf(":%d", nextPort))
			if err != nil {
				nextPort++
				continue
			}
			u.listener = ln
			u.port = nextPort
			used[nextPort] = true
			nextPort++
			break
		}
	}

	return nil
}

// fillSubdirPorts 填充全局子目录端口映射
func fillSubdirPorts(units []*unit, absRoot string) {
	for _, u := range units {
		if u.dir != absRoot {
			subdirPorts[u.name] = u.port
		}
	}
}
