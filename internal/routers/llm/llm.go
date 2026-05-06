// Package llm implements the LLM chat interface and tool executor for gocron.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	llmclient "github.com/ouqiang/gocron/internal/modules/llm"
	"github.com/ouqiang/gocron/internal/models"
	"github.com/ouqiang/gocron/internal/modules/logger"
	"github.com/ouqiang/gocron/internal/modules/utils"
	userAuth "github.com/ouqiang/gocron/internal/routers/user"
	"github.com/ouqiang/gocron/internal/service"
	"gopkg.in/macaron.v1"
)

// ---------- settings helpers (stored in models.Setting table) ----------

const (
	LLMCode         = "llm"
	LLMEndpointKey  = "endpoint"
	LLMAPIKeyKey    = "api_key"
	LLMModelKey     = "model"
	LLMHRModeKey    = "high_risk_mode"
	LLMEnabledKey   = "enabled"

	defaultEndpoint     = "https://api.openai.com/v1"
	defaultModel        = "gpt-4o"
	defaultHighRiskMode = "confirm"
)

// LLMSettings is the serialised form exposed to the API.
type LLMSettings struct {
	Endpoint     string `json:"endpoint"`
	APIKey       string `json:"api_key"`
	Model        string `json:"model"`
	HighRiskMode string `json:"high_risk_mode"` // "confirm" | "auto"
	Enabled      bool   `json:"enabled"`
}

// GetSettings reads LLM settings from the DB.
func GetSettings(ctx *macaron.Context) string {
	jsonResp := utils.JsonResponse{}
	s, err := loadSettings()
	if err != nil {
		logger.Error("load llm settings:", err)
		return jsonResp.CommonFailure("获取LLM配置失败", err)
	}
	// Mask API key in response
	masked := *s
	if masked.APIKey != "" {
		masked.APIKey = "***"
	}
	return jsonResp.Success(utils.SuccessContent, masked)
}

// UpdateSettings writes LLM settings to the DB (admin only).
func UpdateSettings(ctx *macaron.Context) string {
	jsonResp := utils.JsonResponse{}
	endpoint := strings.TrimSpace(ctx.QueryTrim("endpoint"))
	apiKey := strings.TrimSpace(ctx.QueryTrim("api_key"))
	model := strings.TrimSpace(ctx.QueryTrim("model"))
	hrMode := ctx.QueryTrim("high_risk_mode")
	enabled := ctx.QueryTrim("enabled") == "true"

	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	if model == "" {
		model = defaultModel
	}
	if hrMode != "auto" {
		hrMode = "confirm"
	}

	settingModel := new(models.Setting)
	pairs := map[string]string{
		LLMEndpointKey: endpoint,
		LLMModelKey:    model,
		LLMHRModeKey:   hrMode,
	}
	if apiKey != "" && apiKey != "***" {
		pairs[LLMAPIKeyKey] = apiKey
	}
	if enabled {
		pairs[LLMEnabledKey] = "true"
	} else {
		pairs[LLMEnabledKey] = "false"
	}

	for k, v := range pairs {
		if err := upsertSetting(settingModel, LLMCode, k, v); err != nil {
			return jsonResp.CommonFailure("保存LLM配置失败", err)
		}
	}

	return jsonResp.Success("保存成功", nil)
}

// ---------- chat ----------

// ChatRequest is the request body for POST /llm/chat.
type ChatRequest struct {
	Messages       []llmclient.ChatMessage `json:"messages"`
	ConfirmedTools []string                `json:"confirmed_tools"` // high-risk tools the user has pre-confirmed
}

// PendingAction describes a high-risk tool call awaiting user confirmation.
type PendingAction struct {
	Tool        string          `json:"tool"`
	Args        json.RawMessage `json:"args"`
	Description string          `json:"description"`
}

// ChatResponseData is the data field returned by POST /llm/chat.
type ChatResponseData struct {
	Content               string         `json:"content"`
	RequiresConfirmation  bool           `json:"requires_confirmation"`
	PendingAction         *PendingAction `json:"pending_action,omitempty"`
}

// Chat is the main handler for POST /llm/chat.
func Chat(ctx *macaron.Context) string {
	jsonResp := utils.JsonResponse{}

	cfg, err := loadConfig()
	if err != nil || !cfg.Enabled {
		return jsonResp.CommonFailure("LLM功能未启用，请先在设置中配置并启用")
	}

	var req ChatRequest
	if err := ReadBodyJSON(ctx, &req); err != nil || len(req.Messages) == 0 {
		return jsonResp.CommonFailure("请求参数错误：messages不能为空")
	}
	username := userAuth.Username(ctx)
	isAdmin := userAuth.IsAdmin(ctx)

	client := llmclient.NewClient(*cfg)

	// Build conversation: system prompt + user messages
	systemMsg := buildSystemPrompt(username, isAdmin)
	messages := make([]llmclient.ChatMessage, 0, len(req.Messages)+1)
	messages = append(messages, systemMsg)
	messages = append(messages, req.Messages...)

	confirmedSet := make(map[string]bool)
	for _, t := range req.ConfirmedTools {
		confirmedSet[t] = true
	}

	// Tool calling loop (max 6 rounds)
	const maxRounds = 6
	tools := buildTools()
	for round := 0; round < maxRounds; round++ {
		respCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		resp, err := client.Chat(respCtx, messages, tools)
		cancel()
		if err != nil {
			logger.Errorf("LLM chat error: %s", err)
			return jsonResp.CommonFailure("LLM请求失败: " + err.Error())
		}

		if len(resp.Choices) == 0 {
			return jsonResp.CommonFailure("LLM返回空响应")
		}
		choice := resp.Choices[0]

		// No tool calls → final text answer
		if len(choice.Message.ToolCalls) == 0 || choice.FinishReason == "stop" {
			return jsonResp.Success(utils.SuccessContent, ChatResponseData{
				Content: choice.Message.Content,
			})
		}

		// Process each tool call
		messages = append(messages, choice.Message)
		for _, tc := range choice.Message.ToolCalls {
			toolName := tc.Function.Name
			toolArgs := tc.Function.Arguments

			// Check high-risk
			if isHighRisk(toolName) && cfg.HighRiskMode == "confirm" && !confirmedSet[toolName] {
				desc := highRiskDescription(toolName, toolArgs)
				var rawArgs json.RawMessage = json.RawMessage(toolArgs)
				return jsonResp.Success(utils.SuccessContent, ChatResponseData{
					RequiresConfirmation: true,
					PendingAction: &PendingAction{
						Tool:        toolName,
						Args:        rawArgs,
						Description: desc,
					},
				})
			}

			result := executeTool(toolName, toolArgs, username, isAdmin)
			messages = append(messages, llmclient.ChatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       toolName,
				Content:    result,
			})
		}
	}

	return jsonResp.CommonFailure("对话轮次超限，请重新开始")
}

// ---------- internal helpers ----------

func buildSystemPrompt(username string, isAdmin bool) llmclient.ChatMessage {
	role := "普通用户"
	if isAdmin {
		role = "管理员"
	}
	content := fmt.Sprintf(`你是gocron定时任务管理系统的智能助手。
当前登录用户: %s（%s）。
你可以通过工具查看任务列表、节点信息、任务日志，也可以触发任务执行。
请用简洁的中文回复，对数据结果进行摘要说明。
对于高危操作（执行任务、自定义命令），在confirm模式下需要用户确认后才会执行。`, username, role)
	return llmclient.ChatMessage{Role: "system", Content: content}
}

// buildTools returns the tool definitions sent to the LLM.
func buildTools() []llmclient.Tool {
	mustParse := func(s string) json.RawMessage { return json.RawMessage(s) }
	return []llmclient.Tool{
		{
			Type: "function",
			Function: llmclient.ToolFunction{
				Name:        "list_tasks",
				Description: "查询定时任务列表，支持按名称、状态筛选",
				Parameters: mustParse(`{
					"type":"object",
					"properties":{
						"name":{"type":"string","description":"任务名称关键词"},
						"status":{"type":"string","enum":["all","enabled","disabled"],"description":"任务状态，默认all"}
					}
				}`),
			},
		},
		{
			Type: "function",
			Function: llmclient.ToolFunction{
				Name:        "get_task",
				Description: "获取单个任务的详细信息",
				Parameters: mustParse(`{
					"type":"object",
					"properties":{
						"task_id":{"type":"integer","description":"任务ID"}
					},
					"required":["task_id"]
				}`),
			},
		},
		{
			Type: "function",
			Function: llmclient.ToolFunction{
				Name:        "list_hosts",
				Description: "查询所有任务节点（主机）列表",
				Parameters:  mustParse(`{"type":"object","properties":{}}`),
			},
		},
		{
			Type: "function",
			Function: llmclient.ToolFunction{
				Name:        "get_task_logs",
				Description: "获取任务执行日志，支持按任务ID和状态筛选",
				Parameters: mustParse(`{
					"type":"object",
					"properties":{
						"task_id":{"type":"integer","description":"任务ID，0表示所有任务"},
						"status":{"type":"string","enum":["all","running","success","failed"],"description":"日志状态，默认all"},
						"page":{"type":"integer","description":"页码，默认1"}
					}
				}`),
			},
		},
		{
			Type: "function",
			Function: llmclient.ToolFunction{
				Name:        "run_task",
				Description: "⚠️高危操作：立即触发执行指定任务",
				Parameters: mustParse(`{
					"type":"object",
					"properties":{
						"task_id":{"type":"integer","description":"任务ID"}
					},
					"required":["task_id"]
				}`),
			},
		},
		{
			Type: "function",
			Function: llmclient.ToolFunction{
				Name:        "enable_task",
				Description: "启用指定任务",
				Parameters: mustParse(`{
					"type":"object",
					"properties":{
						"task_id":{"type":"integer","description":"任务ID"}
					},
					"required":["task_id"]
				}`),
			},
		},
		{
			Type: "function",
			Function: llmclient.ToolFunction{
				Name:        "disable_task",
				Description: "⚠️高危操作：停用指定任务",
				Parameters: mustParse(`{
					"type":"object",
					"properties":{
						"task_id":{"type":"integer","description":"任务ID"}
					},
					"required":["task_id"]
				}`),
			},
		},
	}
}

var highRiskTools = map[string]bool{
	"run_task":     true,
	"disable_task": true,
}

func isHighRisk(toolName string) bool {
	return highRiskTools[toolName]
}

func highRiskDescription(toolName, argsJSON string) string {
	var args map[string]interface{}
	json.Unmarshal([]byte(argsJSON), &args)
	switch toolName {
	case "run_task":
		return fmt.Sprintf("⚠️ 即将立即执行任务 ID=%v，确认执行吗？", args["task_id"])
	case "disable_task":
		return fmt.Sprintf("⚠️ 即将停用任务 ID=%v，确认停用吗？", args["task_id"])
	default:
		return fmt.Sprintf("⚠️ 即将执行高危操作: %s，参数: %s", toolName, argsJSON)
	}
}

// executeTool runs the named tool and returns a string result.
func executeTool(name, argsJSON, username string, isAdmin bool) string {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "参数解析失败: " + err.Error()
	}

	switch name {
	case "list_tasks":
		return execListTasks(args)
	case "get_task":
		return execGetTask(args)
	case "list_hosts":
		return execListHosts()
	case "get_task_logs":
		return execGetTaskLogs(args)
	case "run_task":
		return execRunTask(args)
	case "enable_task":
		return execEnableTask(args)
	case "disable_task":
		return execDisableTask(args)
	default:
		return fmt.Sprintf("未知工具: %s", name)
	}
}

func intArg(args map[string]interface{}, key string) int {
	v, ok := args[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	}
	return 0
}

func execListTasks(args map[string]interface{}) string {
	taskModel := new(models.Task)
	params := models.CommonMap{
		"Page":     1,
		"PageSize": 20,
	}
	if name, ok := args["name"].(string); ok && name != "" {
		params["Name"] = name
		params["NameMatchType"] = "contains"
	}
	status := -1
	if s, ok := args["status"].(string); ok {
		switch s {
		case "enabled":
			status = int(models.Enabled)
		case "disabled":
			status = int(models.Disabled)
		}
	}
	params["Status"] = status

	tasks, err := taskModel.List(params)
	if err != nil {
		return "查询任务失败: " + err.Error()
	}
	total, _ := taskModel.Total(params)
	result := fmt.Sprintf("共找到 %d 个任务（显示前%d条）：\n", total, len(tasks))
	for _, t := range tasks {
		statusStr := "停用"
		if t.Status == models.Enabled {
			statusStr = "启用"
		}
		result += fmt.Sprintf("- ID:%d 名称:%s 状态:%s Cron:%s\n", t.Id, t.Name, statusStr, t.Spec)
	}
	if len(tasks) == 0 {
		result = "没有找到匹配的任务"
	}
	return result
}

func execGetTask(args map[string]interface{}) string {
	id := intArg(args, "task_id")
	if id <= 0 {
		return "请提供有效的task_id"
	}
	taskModel := new(models.Task)
	task, err := taskModel.Detail(id)
	if err != nil || task.Id == 0 {
		return fmt.Sprintf("未找到任务 ID=%d", id)
	}
	b, _ := json.MarshalIndent(task, "", "  ")
	return string(b)
}

func execListHosts() string {
	hostModel := new(models.Host)
	hosts, err := hostModel.AllList()
	if err != nil {
		return "查询节点失败: " + err.Error()
	}
	if len(hosts) == 0 {
		return "当前没有配置任何任务节点"
	}
	result := fmt.Sprintf("共 %d 个任务节点：\n", len(hosts))
	for _, h := range hosts {
		result += fmt.Sprintf("- ID:%d 名称:%s 端口:%d\n", h.Id, h.Name, h.Port)
	}
	return result
}

func execGetTaskLogs(args map[string]interface{}) string {
	taskLogModel := new(models.TaskLog)
	params := models.CommonMap{
		"Page":     1,
		"PageSize": 20,
	}
	taskID := intArg(args, "task_id")
	if taskID > 0 {
		params["TaskId"] = taskID
	}
	if s, ok := args["status"].(string); ok {
		switch s {
		case "running":
			params["Status"] = int(models.Running)
		case "success":
			params["Status"] = int(models.Finish)
		case "failed":
			params["Status"] = int(models.Failure)
		default:
			params["Status"] = -1
		}
	} else {
		params["Status"] = -1
	}
	if p := intArg(args, "page"); p > 1 {
		params["Page"] = p
	}

	logs, err := taskLogModel.List(params)
	if err != nil {
		return "查询日志失败: " + err.Error()
	}
	total, _ := taskLogModel.Total(params)
	if len(logs) == 0 {
		return "没有找到匹配的任务日志"
	}
	result := fmt.Sprintf("共 %d 条日志（显示前%d条）：\n", total, len(logs))
	for _, l := range logs {
		statusStr := statusLabel(l.Status)
		resultSnippet := l.Result
		if len(resultSnippet) > 100 {
			resultSnippet = resultSnippet[:100] + "..."
		}
		result += fmt.Sprintf("- ID:%d 任务:%s 状态:%s 开始:%s 耗时:%ds\n  输出: %s\n",
			l.Id, l.Name, statusStr,
			l.StartTime.Format("2006-01-02 15:04:05"),
			l.TotalTime,
			resultSnippet)
	}
	return result
}

func execRunTask(args map[string]interface{}) string {
	id := intArg(args, "task_id")
	if id <= 0 {
		return "请提供有效的task_id"
	}
	taskModel := new(models.Task)
	task, err := taskModel.Detail(id)
	if err != nil || task.Id == 0 {
		return fmt.Sprintf("未找到任务 ID=%d", id)
	}
	task.Spec = "手动运行(LLM触发)"
	service.ServiceTask.Run(task)
	return fmt.Sprintf("✅ 任务 '%s'(ID:%d) 已触发执行，请到任务日志中查看结果", task.Name, id)
}

func execEnableTask(args map[string]interface{}) string {
	id := intArg(args, "task_id")
	if id <= 0 {
		return "请提供有效的task_id"
	}
	taskModel := new(models.Task)
	_, err := taskModel.Enable(id)
	if err != nil {
		return "启用失败: " + err.Error()
	}
	// Also re-add to timer
	task, tErr := taskModel.Detail(id)
	if tErr == nil && task.Level == models.TaskLevelParent {
		service.ServiceTask.RemoveAndAdd(task)
	}
	return fmt.Sprintf("✅ 任务 ID=%d 已启用", id)
}

func execDisableTask(args map[string]interface{}) string {
	id := intArg(args, "task_id")
	if id <= 0 {
		return "请提供有效的task_id"
	}
	taskModel := new(models.Task)
	_, err := taskModel.Disable(id)
	if err != nil {
		return "停用失败: " + err.Error()
	}
	service.ServiceTask.Remove(id)
	return fmt.Sprintf("✅ 任务 ID=%d 已停用", id)
}

func statusLabel(s models.Status) string {
	switch s {
	case models.Running:
		return "执行中"
	case models.Finish:
		return "成功"
	case models.Failure:
		return "失败"
	case models.Cancel:
		return "已取消"
	default:
		return "未知"
	}
}

// ---------- DB helpers ----------

func loadSettings() (*LLMSettings, error) {
	list := make([]models.Setting, 0)
	err := models.Db.Where("code = ?", LLMCode).Find(&list)
	if err != nil {
		return nil, err
	}
	s := &LLMSettings{
		Endpoint:     defaultEndpoint,
		Model:        defaultModel,
		HighRiskMode: defaultHighRiskMode,
	}
	for _, v := range list {
		switch v.Key {
		case LLMEndpointKey:
			s.Endpoint = v.Value
		case LLMAPIKeyKey:
			s.APIKey = v.Value
		case LLMModelKey:
			s.Model = v.Value
		case LLMHRModeKey:
			s.HighRiskMode = v.Value
		case LLMEnabledKey:
			s.Enabled = v.Value == "true"
		}
	}
	return s, nil
}

func loadConfig() (*llmclient.Config, error) {
	s, err := loadSettings()
	if err != nil {
		return nil, err
	}
	return &llmclient.Config{
		Endpoint:     s.Endpoint,
		APIKey:       s.APIKey,
		Model:        s.Model,
		HighRiskMode: s.HighRiskMode,
		Enabled:      s.Enabled,
	}, nil
}

func upsertSetting(m *models.Setting, code, key, value string) error {
	existing := new(models.Setting)
	has, err := models.Db.Where("code = ? AND key = ?", code, key).Get(existing)
	if err != nil {
		return err
	}
	if has {
		_, err = models.Db.ID(existing.Id).Cols("value").Update(&models.Setting{Value: value})
		return err
	}
	_, err = models.Db.Insert(&models.Setting{Code: code, Key: key, Value: value})
	return err
}

// InitLLMSettings inserts default LLM settings into the DB if they don't exist.
func InitLLMSettings() {
	if models.Db == nil {
		return
	}
	defaults := map[string]string{
		LLMEndpointKey: defaultEndpoint,
		LLMAPIKeyKey:   "",
		LLMModelKey:    defaultModel,
		LLMHRModeKey:   defaultHighRiskMode,
		LLMEnabledKey:  "false",
	}
	for k, v := range defaults {
		existing := new(models.Setting)
		has, _ := models.Db.Where("code = ? AND key = ?", LLMCode, k).Get(existing)
		if !has {
			models.Db.Insert(&models.Setting{Code: LLMCode, Key: k, Value: v})
		}
	}
}

// ReadBodyJSON is a helper that reads and decodes JSON from the macaron request body.
func ReadBodyJSON(ctx *macaron.Context, v interface{}) error {
	rb := ctx.Req.Body()
	if rb == nil {
		return json.Unmarshal([]byte("{}"), v)
	}
	rc := rb.ReadCloser()
	if rc == nil {
		return json.Unmarshal([]byte("{}"), v)
	}
	defer rc.Close()

	var body []byte
	buf := make([]byte, 512)
	for {
		n, err := rc.Read(buf)
		body = append(body, buf[:n]...)
		if err != nil {
			break
		}
	}
	if len(body) == 0 {
		return json.Unmarshal([]byte("{}"), v)
	}
	return json.Unmarshal(body, v)
}
