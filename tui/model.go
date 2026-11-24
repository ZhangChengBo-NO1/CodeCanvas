package tui

import (
	"context"
	"strings"

	"CodeCanvas/internal/loop"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------- 自定义消息 ----------

// AgentResponseMsg 是后台 Agent 执行完毕后发送给 TUI 的消息
type AgentResponseMsg struct {
	Reply string
	Err   error
}

// ---------- TUI Model ----------

type Model struct {
	viewport  viewport.Model
	textInput textinput.Model

	// 状态
	ready     bool
	follow    bool
	isLoading bool // 是否正在等待大模型回复

	// 历史消息（纯文本，用于渲染）
	messages []string

	// 依赖注入
	agent   *loop.Agent
	session *loop.Session
}

// NewModel 接收 Agent 和 Session 进行初始化
func NewModel(agent *loop.Agent, session *loop.Session) Model {
	ti := textinput.New()
	ti.Placeholder = "输入你的需求..."
	ti.Focus()
	ti.Width = 80

	return Model{
		textInput: ti,
		messages:  []string{"🤖 CodeCanvas TUI 已启动。输入内容开始对话。"},
		follow:    true,
		agent:     agent,
		session:   session,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// ---------- Update ----------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// 计算 viewport 的高度，确保不会因为终端窗口太小而变成负数
		vpHeight := msg.Height - 4
		if vpHeight < 1 {
			vpHeight = 1
		}

		if !m.ready {
			m.viewport = viewport.New(msg.Width, vpHeight)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = vpHeight
		}
		m.textInput.Width = msg.Width - 6
		
		// 只有在尺寸稳定后，才同步内容
		m.syncViewportContent()

	case tea.KeyMsg:
		// 如果正在加载，禁止发送新消息
		if m.isLoading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "enter":
			input := strings.TrimSpace(m.textInput.Value())
			if input != "" {
				// 1. 把用户消息加到界面
				m.messages = append(m.messages, "🧑 你: "+input)
				m.textInput.Reset()
				m.isLoading = true
				m.follow = true
				m.syncViewportContent()

				// 2. 启动后台 Goroutine 异步调用 Agent
				return m, func() tea.Msg {
					// 注意：这里不能直接用 m.agent.Run，因为 m 是值拷贝
					// 但 Agent 的指针是安全的
					reply, err := m.agent.Run(context.Background(), m.session, input)
					return AgentResponseMsg{Reply: reply, Err: err}
				}
			}

		case "up", "k", "pgup":
			m.follow = false
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)

		case "down", "j", "pgdown":
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
			if m.viewport.AtBottom() {
				m.follow = true
			}
		}

		// 🚀 修复点：无论上面是否匹配到了特殊按键，只要没 return 退出，
		// 最后都让 textInput 处理一次，这样普通字母和数字就能正常输入了。
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)

	case AgentResponseMsg:
		m.isLoading = false
		if msg.Err != nil {
			m.messages = append(m.messages, "❌ 错误: "+msg.Err.Error())
		} else {
			m.messages = append(m.messages, "🤖 Agent: "+msg.Reply)
		}
		m.syncViewportContent()

	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) syncViewportContent() {
	// 如果 viewport 还没初始化好，直接返回
	if !m.ready {
		return
	}

	content := strings.Join(m.messages, "\n\n")
	if m.isLoading {
		content += "\n\n⏳ Agent 正在思考中..."
	}
	
	m.viewport.SetContent(content)
	
	// 防止在空内容或高度为 0 时触发 GotoBottom 内部越界
	if m.follow && m.viewport.Height > 0 && len(content) > 0 {
		m.viewport.GotoBottom()
	}
}

// ---------- View ----------

var (
	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

func (m Model) View() string {
	// 如果还没收到窗口尺寸，或者尺寸无效，不渲染
	if !m.ready || m.viewport.Width <= 0 || m.viewport.Height <= 0 {
		return "Initializing..."
	}

	header := m.viewport.View()
	input := inputStyle.Render(m.textInput.View())
	
	help := "↑/↓ 滚动 • Enter 发送 • Esc 退出"
	if m.isLoading {
		help = "⏳ 请稍候，Agent 正在执行..."
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		input,
		helpStyle.Render(help),
	)
}