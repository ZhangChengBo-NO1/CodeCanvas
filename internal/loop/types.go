// internal/loop/types.go
package loop

import (
	"context"
	"encoding/json"
)

// Tool 是 Agent 能调用的一个能力。
// 任何"能力"——Shell、write_file、MCP 工具、echo——都实现这个接口。
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any // JSON Schema
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}