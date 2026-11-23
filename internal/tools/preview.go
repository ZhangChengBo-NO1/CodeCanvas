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
	// 如果服务器已经在运行，直接返回 URL
	if p.URL != "" {
		return fmt.Sprintf("预览服务器已在运行: %s", p.URL), nil
	}

	// 动态分配一个空闲端口（监听 0 端口）
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("启动端口失败: %w", err)
	}
	p.URL = "http://" + ln.Addr().String()

	// 挂载静态文件服务
	p.srv = &http.Server{
		Handler: http.FileServer(http.Dir(p.Root)),
	}

	// 后台协程运行服务器
	go func() {
		p.srv.Serve(ln)
	}()

	return fmt.Sprintf("✅ 预览服务已启动。用户可以通过浏览器访问: %s", p.URL), nil
}

func (p *PreviewServer) Shutdown(ctx context.Context) error {
	if p.srv != nil {
		return p.srv.Shutdown(ctx)
	}
	return nil
}