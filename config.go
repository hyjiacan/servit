package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type meta struct {
	title string
	desc  string
	port  int
}

// readMeta 读取目录下的 servit.conf，解析 title / desc / port。
// 不存在或解析失败时返回零值。
func readMeta(dir string) meta {
	var m meta
	f, err := os.Open(filepath.Join(dir, "servit.conf"))
	if err != nil {
		return m
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "title":
			m.title = val
		case "desc":
			m.desc = val
		case "port":
			if p, err := strconv.Atoi(val); err == nil && p >= 1 && p <= 65535 {
				m.port = p
			}
		}
	}
	return m
}
