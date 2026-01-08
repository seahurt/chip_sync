<template>
  <div class="log-viewer">
    <div class="log-header">
      <h3><el-icon><Document /></el-icon> 同步日志</h3>
      <el-button size="small" text @click="$emit('refresh')">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>
    <div class="log-content" ref="logContainer">
      <div 
        v-for="(log, index) in logs" 
        :key="index" 
        class="log-line"
        :class="getLogClass(log)"
      >
        {{ log }}
      </div>
      <div v-if="logs.length === 0" class="log-empty">
        暂无日志
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick, defineProps, defineEmits } from 'vue'

const props = defineProps({
  logs: {
    type: Array,
    default: () => []
  }
})

defineEmits(['refresh'])

const logContainer = ref(null)

// 自动滚动到底部
watch(() => props.logs.length, async () => {
  await nextTick()
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
})

const getLogClass = (log) => {
  if (log.includes('ERROR') || log.includes('错误') || log.includes('失败')) {
    return 'log-error'
  }
  if (log.includes('WARN') || log.includes('警告')) {
    return 'log-warn'
  }
  if (log.includes('成功') || log.includes('完成')) {
    return 'log-success'
  }
  return ''
}
</script>

<style scoped>
.log-viewer {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--card-bg);
  border-radius: 12px;
  border: 1px solid var(--border-color);
  overflow: hidden;
}

.log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
}

.log-header h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.log-content {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  font-family: 'Cascadia Code', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.6;
}

.log-line {
  padding: 2px 8px;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-line:hover {
  background: rgba(255, 255, 255, 0.05);
}

.log-error {
  color: var(--danger-color);
}

.log-warn {
  color: var(--warning-color);
}

.log-success {
  color: var(--success-color);
}

.log-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-secondary);
}
</style>
