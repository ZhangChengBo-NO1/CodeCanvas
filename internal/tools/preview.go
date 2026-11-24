package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

type PreviewServer struct {
	Root string
	srv  *http.Server
	URL  string
}

func (p *PreviewServer) Name() string { return "start_preview" }
func (p *PreviewServer) Description() string {
	return "启动本地静态预览服务器，将当前工作区的文件暴露在网页上。如果已经启动则返回现有 URL。"
}
func (p *PreviewServer) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{}, // 不需要参数
	}
}

func (p *PreviewServer) Execute(ctx context.Context, raw json.RawMessage) (string, error) {
	if p.URL != "" {
		return fmt.Sprintf("预览服务器已在运行: %s", p.URL), nil
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("启动端口失败: %w", err)
	}
	p.URL = "http://" + ln.Addr().String()

	// 不要在这里打印任何东西！
	fs := http.FileServer(http.Dir(p.Root))
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r) // 纯静默处理，不要 fmt.Printf
	})

	p.srv = &http.Server{Handler: mux}


	go func() {
		if err := p.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Printf("❌ [预览服务器] 异常退出: %v\n", err)
		}
	}()

	return fmt.Sprintf("✅ 预览服务已启动。用户可以通过浏览器访问: %s", p.URL), nil
}

func (p *PreviewServer) Shutdown(ctx context.Context) error {
	if p.srv != nil {
		return p.srv.Shutdown(ctx)
	}
	return nil
}