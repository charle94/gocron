<template>
  <div>
    <el-row>
      <el-col :span="12" :offset="6">
        <el-card>
          <div slot="header">
            <span>AI 助手配置</span>
          </div>
          <el-form :model="form" :rules="rules" ref="settingsForm" label-width="120px">
            <el-form-item label="启用AI助手">
              <el-switch v-model="form.enabled"></el-switch>
            </el-form-item>
            <el-form-item label="API 端点" prop="endpoint">
              <el-input v-model.trim="form.endpoint" placeholder="https://api.openai.com/v1"></el-input>
              <div class="form-tip">支持任意 OpenAI 兼容端点（如 Ollama、Azure OpenAI 等）</div>
            </el-form-item>
            <el-form-item label="API Key">
              <el-input v-model="form.api_key" type="password" placeholder="留空表示不修改" show-password></el-input>
            </el-form-item>
            <el-form-item label="模型" prop="model">
              <el-input v-model.trim="form.model" placeholder="gpt-4o"></el-input>
            </el-form-item>
            <el-form-item label="高危操作模式">
              <el-radio-group v-model="form.high_risk_mode">
                <el-radio label="confirm">确认模式（执行高危操作前需用户确认）</el-radio>
                <el-radio label="auto">自动模式（直接执行，谨慎使用）</el-radio>
              </el-radio-group>
              <div class="form-tip">高危操作包括：立即执行任务、停用任务等</div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="saving" @click="save">保存</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script>
import llmService from '../../api/llm'

export default {
  name: 'LLMSettings',
  data () {
    return {
      form: {
        enabled: false,
        endpoint: 'https://api.openai.com/v1',
        api_key: '',
        model: 'gpt-4o',
        high_risk_mode: 'confirm'
      },
      saving: false,
      rules: {
        endpoint: [{ required: true, message: '请输入API端点', trigger: 'blur' }],
        model: [{ required: true, message: '请输入模型名称', trigger: 'blur' }]
      }
    }
  },
  created () {
    this.loadSettings()
  },
  methods: {
    loadSettings () {
      llmService.getSettings((data) => {
        if (!data) return
        this.form.enabled = data.enabled || false
        this.form.endpoint = data.endpoint || 'https://api.openai.com/v1'
        this.form.model = data.model || 'gpt-4o'
        this.form.high_risk_mode = data.high_risk_mode || 'confirm'
        // Don't overwrite api_key - server returns '***' for masked value
      })
    },
    save () {
      this.$refs.settingsForm.validate((valid) => {
        if (!valid) return
        this.saving = true
        llmService.updateSettings({
          enabled: this.form.enabled ? 'true' : 'false',
          endpoint: this.form.endpoint,
          api_key: this.form.api_key,
          model: this.form.model,
          high_risk_mode: this.form.high_risk_mode
        }, () => {
          this.saving = false
          this.$message.success('保存成功')
          this.form.api_key = ''
        })
      })
    }
  }
}
</script>

<style scoped>
.form-tip {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
  margin-top: 4px;
}
</style>
