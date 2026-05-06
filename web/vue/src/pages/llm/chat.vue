<template>
  <div class="llm-chat">
    <el-row :gutter="10">
      <el-col :span="18" :offset="3">
        <el-card>
          <div slot="header" class="chat-header">
            <span><i class="el-icon-chat-dot-square"></i> AI 智能助手</span>
            <el-button type="text" size="small" style="float:right" @click="clearChat">清空对话</el-button>
          </div>

          <!-- 消息列表 -->
          <div class="chat-messages" ref="messageContainer">
            <div v-if="messages.length === 0" class="chat-empty">
              <p>🤖 您好！我是gocron智能助手，可以帮您：</p>
              <ul>
                <li>📋 查看任务列表和详情</li>
                <li>🖥️ 查看任务节点信息</li>
                <li>📝 查看和分析任务执行日志</li>
                <li>▶️ 触发执行任务（高危操作需确认）</li>
                <li>🔛 启用 / 停用任务</li>
              </ul>
              <p>请输入您的问题或指令…</p>
            </div>

            <div v-for="(msg, idx) in messages" :key="idx" :class="['message', msg.role]">
              <div class="message-bubble">
                <div class="message-role">{{ roleLabel(msg.role) }}</div>
                <div class="message-content" v-html="formatContent(msg.content)"></div>
              </div>
            </div>

            <!-- 高危操作确认卡片 -->
            <div v-if="pendingAction" class="message assistant">
              <div class="message-bubble warning">
                <div class="message-role">⚠️ 高危操作确认</div>
                <div class="message-content">{{ pendingAction.description }}</div>
                <div style="margin-top:10px">
                  <el-button type="danger" size="small" @click="confirmAction">确认执行</el-button>
                  <el-button size="small" @click="cancelAction">取消</el-button>
                </div>
              </div>
            </div>

            <!-- 加载中 -->
            <div v-if="loading" class="message assistant">
              <div class="message-bubble">
                <div class="message-role">AI 助手</div>
                <div class="message-content"><i class="el-icon-loading"></i> 思考中…</div>
              </div>
            </div>
          </div>

          <!-- 输入区 -->
          <div class="chat-input">
            <el-input
              v-model="inputText"
              type="textarea"
              :rows="2"
              placeholder="输入消息，Ctrl+Enter 发送"
              :disabled="loading || !!pendingAction"
              @keydown.ctrl.enter.native="sendMessage">
            </el-input>
            <div class="chat-input-actions">
              <el-button
                type="primary"
                :loading="loading"
                :disabled="!inputText.trim() || !!pendingAction"
                @click="sendMessage">
                发送
              </el-button>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script>
import llmService from '../../api/llm'

export default {
  name: 'LLMChat',
  data () {
    return {
      messages: [], // {role, content} shown in UI
      apiMessages: [], // messages sent to backend (full history)
      inputText: '',
      loading: false,
      pendingAction: null, // {tool, args, description}
      pendingConfirmedTools: []
    }
  },
  methods: {
    roleLabel (role) {
      const map = { user: '我', assistant: 'AI 助手', tool: '工具结果', system: '系统' }
      return map[role] || role
    },

    formatContent (text) {
      if (!text) return ''
      return text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/\n/g, '<br>')
        .replace(/`([^`]+)`/g, '<code>$1</code>')
    },

    scrollToBottom () {
      this.$nextTick(() => {
        const el = this.$refs.messageContainer
        if (el) el.scrollTop = el.scrollHeight
      })
    },

    clearChat () {
      this.messages = []
      this.apiMessages = []
      this.pendingAction = null
      this.pendingConfirmedTools = []
    },

    sendMessage () {
      const text = this.inputText.trim()
      if (!text || this.loading) return

      this.inputText = ''
      const userMsg = { role: 'user', content: text }
      this.messages.push(userMsg)
      this.apiMessages.push(userMsg)
      this.scrollToBottom()
      this.callLLM()
    },

    callLLM (confirmedTools) {
      this.loading = true
      const payload = {
        messages: this.apiMessages,
        confirmed_tools: confirmedTools || []
      }
      llmService.chat(payload, (data) => {
        this.loading = false
        if (!data) return

        if (data.requires_confirmation && data.pending_action) {
          this.pendingAction = data.pending_action
          this.scrollToBottom()
          return
        }

        if (data.content) {
          const assistantMsg = { role: 'assistant', content: data.content }
          this.messages.push(assistantMsg)
          this.apiMessages.push(assistantMsg)
          this.scrollToBottom()
        }
      })
    },

    confirmAction () {
      if (!this.pendingAction) return
      const toolName = this.pendingAction.tool
      this.pendingAction = null
      this.pendingConfirmedTools.push(toolName)
      this.callLLM(this.pendingConfirmedTools)
    },

    cancelAction () {
      const msg = { role: 'assistant', content: '操作已取消。' }
      this.messages.push(msg)
      this.apiMessages.push(msg)
      this.pendingAction = null
      this.scrollToBottom()
    }
  }
}
</script>

<style scoped>
.llm-chat {
  padding: 20px 0;
}
.chat-header {
  font-size: 16px;
  font-weight: bold;
}
.chat-messages {
  min-height: 400px;
  max-height: 500px;
  overflow-y: auto;
  padding: 10px;
  background: #f5f7fa;
  border-radius: 4px;
  margin-bottom: 12px;
}
.chat-empty {
  color: #909399;
  padding: 20px;
  line-height: 2;
}
.chat-empty ul {
  padding-left: 20px;
}
.message {
  margin-bottom: 16px;
  display: flex;
}
.message.user {
  justify-content: flex-end;
}
.message.assistant,
.message.tool,
.message.system {
  justify-content: flex-start;
}
.message-bubble {
  max-width: 75%;
  padding: 10px 14px;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  word-break: break-word;
}
.message.user .message-bubble {
  background: #409eff;
  color: #fff;
}
.message-bubble.warning {
  background: #fdf6ec;
  border: 1px solid #e6a23c;
}
.message-role {
  font-size: 11px;
  color: #c0c4cc;
  margin-bottom: 4px;
}
.message.user .message-role {
  color: rgba(255,255,255,0.7);
}
.message-content {
  font-size: 14px;
  line-height: 1.6;
}
.message-content code {
  background: #f0f0f0;
  padding: 0 4px;
  border-radius: 3px;
  font-family: monospace;
}
.chat-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.chat-input-actions {
  display: flex;
  justify-content: flex-end;
}
</style>
