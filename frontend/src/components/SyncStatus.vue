<template>
  <div class="sync-status" :class="statusClass">
    <div class="status-indicator">
      <div class="status-dot" :class="{ 'pulse': isRunning }"></div>
      <span class="status-text">{{ statusText }}</span>
    </div>
    
    <div class="status-details" v-if="lastResult">
      <div class="detail-item">
        <el-icon><Clock /></el-icon>
        <span>{{ formatTime(lastResult.end_time) }}</span>
      </div>
      <div class="detail-item" v-if="lastResult.error">
        <el-icon><Warning /></el-icon>
        <span class="error-text">{{ lastResult.error }}</span>
      </div>
    </div>

    <div class="status-actions">
      <el-button 
        type="primary" 
        size="small" 
        :loading="isRunning"
        @click="$emit('sync')"
      >
        <el-icon><Refresh /></el-icon>
        {{ isRunning ? '同步中...' : '立即同步' }}
      </el-button>
      
      <el-button 
        :type="schedulerRunning ? 'danger' : 'success'" 
        size="small"
        @click="$emit('toggleScheduler')"
      >
        <el-icon>
          <VideoPause v-if="schedulerRunning" />
          <VideoPlay v-else />
        </el-icon>
        {{ schedulerRunning ? '停止定时' : '启动定时' }}
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { computed, defineProps, defineEmits } from 'vue'

const props = defineProps({
  status: {
    type: Object,
    default: () => ({})
  }
})

defineEmits(['sync', 'toggleScheduler'])

const isRunning = computed(() => props.status?.isRunning || false)
const schedulerRunning = computed(() => props.status?.schedulerRunning || false)
const lastResult = computed(() => props.status?.lastResult)

const statusText = computed(() => {
  if (isRunning.value) return '同步中'
  if (props.status?.isDirty) return '等待同步'
  return props.status?.status || '空闲'
})

const statusClass = computed(() => {
  if (isRunning.value) return 'status-running'
  if (props.status?.status === '错误') return 'status-error'
  return 'status-idle'
})

const formatTime = (timeStr) => {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN')
}
</script>

<style scoped>
.sync-status {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 16px;
  padding: 16px 20px;
  background: var(--card-bg);
  border-radius: 12px;
  border: 1px solid var(--border-color);
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--info-color);
}

.status-idle .status-dot {
  background: var(--success-color);
}

.status-running .status-dot {
  background: var(--primary-color);
}

.status-error .status-dot {
  background: var(--danger-color);
}

.status-dot.pulse {
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(1.2); }
}

.status-text {
  font-size: 16px;
  font-weight: 600;
}

.status-details {
  display: flex;
  gap: 16px;
  color: var(--text-secondary);
  font-size: 12px;
}

.detail-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.error-text {
  color: var(--danger-color);
}

.status-actions {
  display: flex;
  gap: 8px;
}
</style>
