<template>
  <div class="config-panel">
    <el-form :model="config" label-width="120px" label-position="top">
      <!-- Rsync 设置 -->
      <div class="form-section">
        <h3><el-icon><Setting /></el-icon> Rsync 配置</h3>
        
        <el-form-item label="Rsync 路径">
          <el-input v-model="config.rsync_path" placeholder="rsync 可执行文件路径">
            <template #prepend>
              <el-icon><Document /></el-icon>
            </template>
          </el-input>
        </el-form-item>
      </div>

      <!-- 远程服务器设置 -->
      <div class="form-section">
        <h3><el-icon><Connection /></el-icon> 远程服务器</h3>
        
        <el-row :gutter="16">
          <el-col :span="16">
            <el-form-item label="服务器地址">
              <el-input v-model="config.remote_host" placeholder="hostname 或 IP" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="端口">
              <el-input-number v-model="config.remote_port" :min="1" :max="65535" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="模块名">
          <el-input v-model="config.remote_module" placeholder="rsync 模块名" />
        </el-form-item>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="用户名">
              <el-input v-model="config.username" placeholder="认证用户名" />
            </el-form-item>
          </el-col>
            <el-col :span="12">
              <el-form-item label="密码">
                <el-input v-model="config.password" placeholder="密码" type="password" />
              </el-form-item>
            </el-col>
        </el-row>
      </div>

      <!-- 同步设置 -->
      <div class="form-section">
        <h3><el-icon><Timer /></el-icon> 同步设置</h3>
        
        <el-form-item label="同步间隔（秒）">
          <el-slider 
            v-model="config.sync_interval_seconds" 
            :min="30" 
            :max="3600" 
            :step="30"
            show-input
          />
        </el-form-item>
      </div>

      <!-- 操作按钮 -->
      <div class="form-actions">
        <el-button type="primary" @click="saveConfig" :loading="saving">
          <el-icon><Check /></el-icon>
          保存配置
        </el-button>
      </div>
    </el-form>
  </div>
</template>

<script setup>
import { reactive, ref, watch, defineProps, defineEmits } from 'vue'

const props = defineProps({
  modelValue: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['update:modelValue', 'save'])

const config = reactive({ ...props.modelValue })
const saving = ref(false)

watch(() => props.modelValue, (newVal) => {
  Object.assign(config, newVal)
})

watch(config, (newVal) => {
  emit('update:modelValue', newVal)
})

const saveConfig = async () => {
  saving.value = true
  try {
    emit('save', config)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.config-panel {
  height: 100%;
  overflow-y: auto;
  padding: 16px;
}

.form-section {
  background: rgba(255, 255, 255, 0.03);
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 16px;
  border: 1px solid var(--border-color);
}

.form-section h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  color: var(--primary-color);
  font-size: 14px;
  font-weight: 600;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
