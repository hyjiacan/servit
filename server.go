package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// buildHandlers 为每个单元创建 http.Server
func buildHandlers(units []*unit, absRoot string, spa, cors bool) {
	for _, u := range units {
		var handler http.Handler
		if u.dir == absRoot {
			handler = rootHandler(absRoot, spa)
		} else {
			handler = http.FileServer(http.Dir(u.dir))
			if spa {
				handler = spaFallback(u.dir, handler)
			}
		}
		if cors {
			handler = corsMiddleware(handler)
		}
		handler = logMiddleware(u.port, handler)

		u.server = &http.Server{Handler: handler}
	}
}

// ---------- 根目录 handler ----------

func rootHandler(rootDir string, spa bool) http.Handler {
	fileServer := http.FileServer(http.Dir(rootDir))
	if spa {
		fileServer = spaFallback(rootDir, fileServer)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			renderIndex(w, rootDir)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func renderIndex(w http.ResponseWriter, rootDir string) {
	subdirs, _ := scanSubdirs(rootDir)
	lan := lanIP()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Servit 索引</title>
<style>
body{font-family:-apple-system,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;background:#f6f7f9;color:#24292f;line-height:1.7;margin:0}
main{max-width:720px;margin:0 auto;padding:48px 24px 80px}
h1{font-size:26px;margin-top:0}
p{color:#57606a}
ul{list-style:none;padding:0}
li{background:#fff;border:1px solid #d0d7de;border-radius:10px;margin-bottom:10px;padding:16px 20px}
.name{font-size:17px;font-weight:600;display:block}
.desc{color:#57606a;font-size:14px;margin-top:2px;display:block}
.addr{display:block;font-size:12px;margin-top:6px;font-family:ui-monospace,SFMono-Regular,Consolas,monospace}
.addr a{color:#0969da;text-decoration:none}
.addr a:hover{text-decoration:underline}
.empty{background:#fff;border:1px dashed #d0d7de;border-radius:10px;padding:24px;color:#57606a}
code{background:#eef0f3;padding:2px 6px;border-radius:4px}
</style>
</head>
<body>
<main>
<h1>🪨 Servit 索引</h1>
<p>以下是当前所有子目录，各自独立端口访问。</p>
`)

	if len(subdirs) == 0 {
		fmt.Fprint(w, `<div class="empty">暂无子目录。在 <code>www/</code> 下新建一个文件夹，刷新即可看到。</div>`)
	} else {
		fmt.Fprint(w, "<ul>\n")
		for _, d := range subdirs {
			name := baseName(d)
			m := readMeta(d)
			title := m.title
			if title == "" {
				title = name
			}
			port := subdirPorts[name]

			fmt.Fprintf(w, `<li>
<span class="name">%s</span>`, htmlEscape(title))
			if m.desc != "" {
				fmt.Fprintf(w, `<span class="desc">%s</span>`, htmlEscape(m.desc))
			}

			fmt.Fprintf(w, `<span class="addr"><a href="http://localhost:%d/">http://localhost:%d</a></span>`, port, port)
			fmt.Fprintf(w, `<span class="addr"><a href="http://127.0.0.1:%d/">http://127.0.0.1:%d</a></span>`, port, port)
			if lan != "" {
				fmt.Fprintf(w, `<span class="addr"><a href="http://%s:%d/">http://%s:%d</a></span>`, lan, port, lan, port)
			}

			fmt.Fprint(w, `</li>`+"\n")
		}
		fmt.Fprint(w, "</ul>\n")
	}

	fmt.Fprint(w, `</main>
</body>
</html>`)
}

func htmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return r.Replace(s)
}

// ---------- 中间件 ----------

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func logMiddleware(port int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		fmt.Printf("[%d] %s %s %d %s\n",
			port, r.Method, r.URL.Path, rec.status,
			time.Since(start).Round(time.Millisecond))
	})
}

// ---------- SPA 回退 ----------

func spaFallback(root string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(root, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			http.ServeFile(w, r, filepath.Join(root, "index.html"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
