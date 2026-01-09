<template>
  <div class="app-container">
    <!-- 顶部标题栏 -->
    <header class="app-header">
      <div class="brand">
        <div class="logo">
          <el-icon :size="28">
            <Connection />
          </el-icon>
        </div>
        <div class="title-group">
          <h1>ChipSync Windows</h1>
          <span class="subtitle">测序芯片数据同步工具</span>
        </div>
      </div>
    </header>

    <!-- 同步状态栏 -->
    <section class="status-section">
      <SyncStatus :status="syncStatus" @sync="handleSync" @toggle-scheduler="handleToggleScheduler" />
    </section>

    <!-- 主内容区域 -->
    <main class="main-content">
      <!-- 左侧配置面板 -->
      <aside class="config-section">
        <ConfigPanel v-model="config" @save="handleSaveConfig" />
      </aside>

      <!-- 右侧区域：目录 + 日志 -->
      <section class="content-section">
        <!-- 目录选择 -->
        <div class="directory-panel">
          <div class="panel-header">
            <h3><el-icon>
                <Folder />
              </el-icon> 芯片目录</h3>
            <div class="panel-actions">
              <el-button size="small" @click="fetchChipDirs" :loading="refreshingDirs">
                <el-icon>
                  <Refresh />
                </el-icon>
                刷新状态
              </el-button>
              <el-button type="primary" size="small" @click="handleSelectDirectory">
                <el-icon>
                  <FolderOpened />
                </el-icon>
                选择目录
              </el-button>
            </div>
          </div>

          <div class="directory-path" v-if="config.local_path">
            <el-icon>
              <FolderOpened />
            </el-icon>
            <span>{{ config.local_path }}</span>
          </div>
          <div class="directory-path empty" v-else>
            请选择芯片父目录
          </div>

          <div class="chip-list" v-if="chipDirs.length > 0">
            <div class="chip-item" :class="{ 'chip-stable': dir.is_stable }" v-for="dir in chipDirs" :key="dir.name"
              :title="dir.last_modified ? `最后修改: ${dir.last_modified}` : ''">
              <el-icon>
                <Folder />
              </el-icon>
              <span>{{ dir.name }}</span>
              <el-tag :type="dir.is_stable ? 'info' : 'success'" size="small" effect="plain">
                {{ dir.is_stable ? '已完成' : '同步中' }}
              </el-tag>
            </div>
          </div>
        </div>

        <!-- 日志查看器 -->
        <div class="log-section">
          <LogViewer :logs="logs" @refresh="fetchLogs" />
        </div>
      </section>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import ConfigPanel from './components/ConfigPanel.vue'
import SyncStatus from './components/SyncStatus.vue'
import LogViewer from './components/LogViewer.vue'
import { GetConfig, SaveConfig, SelectDirectory, StartSync, GetSyncStatus, GetLogs, StartScheduler, StopScheduler, GetChipDirs } from '../wailsjs/go/main/App'
import { ElMessage } from 'element-plus'

// 配置
const config = ref({
  rsync_path: 'rsync',
  remote_host: '',
  remote_port: 873,
  remote_module: '',
  username: '',
  password: '',
  local_path: '',
  sync_interval_seconds: 300,
  stable_hours: 12,
  log_path: ''
})

// 同步状态
const syncStatus = ref({})

// 芯片目录列表
const chipDirs = ref([])

// 日志
const logs = ref([])

// 刷新状态
const refreshingDirs = ref(false)

// 定时器
let statusTimer = null
let logsTimer = null
let chipDirsTimer = null

// 加载配置
const fetchConfig = async () => {
  try {
    const cfg = await GetConfig()
    config.value = cfg
  } catch (e) {
    console.error('获取配置失败:', e)
  }
}

// 保存配置
const handleSaveConfig = async (cfg) => {
  try {
    await SaveConfig(cfg)
    ElMessage.success('配置已保存')
    fetchChipDirs()
  } catch (e) {
    ElMessage.error('保存配置失败: ' + e)
  }
}

// 选择目录
const handleSelectDirectory = async () => {
  try {
    const dir = await SelectDirectory()
    if (dir) {
      config.value.local_path = dir
      await handleSaveConfig(config.value)
    }
  } catch (e) {
    ElMessage.error('选择目录失败: ' + e)
  }
}

// 手动同步
const handleSync = async () => {
  try {
    await StartSync()
    ElMessage.info('同步任务已启动')
    fetchStatus()
  } catch (e) {
    ElMessage.error('启动同步失败: ' + e)
  }
}

// 切换调度器
const handleToggleScheduler = async () => {
  try {
    if (syncStatus.value.schedulerRunning) {
      await StopScheduler()
      ElMessage.info('定时同步已停止')
    } else {
      await StartScheduler()
      ElMessage.success('定时同步已启动')
    }
    fetchStatus()
  } catch (e) {
    ElMessage.error('操作失败: ' + e)
  }
}

// 获取同步状态
const fetchStatus = async () => {
  try {
    syncStatus.value = await GetSyncStatus()
  } catch (e) {
    console.error('获取状态失败:', e)
  }
}

// 获取日志
const fetchLogs = async () => {
  try {
    logs.value = await GetLogs()
  } catch (e) {
    console.error('获取日志失败:', e)
  }
}

// 获取芯片目录
const fetchChipDirs = async () => {
  refreshingDirs.value = true
  try {
    chipDirs.value = await GetChipDirs()
  } catch (e) {
    console.error('获取芯片目录失败:', e)
  } finally {
    refreshingDirs.value = false
  }
}

// 启动定时刷新
const startPolling = () => {
  statusTimer = setInterval(fetchStatus, 2000)
  logsTimer = setInterval(fetchLogs, 3000)
  chipDirsTimer = setInterval(fetchChipDirs, 10000) // 每10秒刷新目录状态
}

onMounted(() => {
  fetchConfig()
  fetchStatus()
  fetchLogs()
  fetchChipDirs()
  startPolling()
})

onUnmounted(() => {
  if (statusTimer) clearInterval(statusTimer)
  if (logsTimer) clearInterval(logsTimer)
  if (chipDirsTimer) clearInterval(chipDirsTimer)
})
</script>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

/* 顶部标题栏 */
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 24px;
  background: linear-gradient(135deg, var(--card-bg) 0%, rgba(22, 33, 62, 0.8) 100%);
  border-bottom: 1px solid var(--border-color);
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background: linear-gradient(135deg, var(--primary-color), #67c23a);
  border-radius: 12px;
  color: white;
}

.title-group h1 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  background: linear-gradient(90deg, #fff 0%, var(--primary-color) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.subtitle {
  font-size: 12px;
  color: var(--text-secondary);
}

/* 状态栏 */
.status-section {
  padding: 16px 24px;
}

/* 主内容 */
.main-content {
  flex: 1;
  display: flex;
  gap: 16px;
  padding: 0 24px 24px;
  overflow: hidden;
}

.config-section {
  width: 380px;
  flex-shrink: 0;
  overflow: hidden;
}

.content-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow: hidden;
}

/* 目录面板 */
.directory-panel {
  background: var(--card-bg);
  border-radius: 12px;
  border: 1px solid var(--border-color);
  padding: 16px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.panel-header h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.panel-actions {
  display: flex;
  gap: 8px;
}

.directory-path {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: var(--bg-color);
  border-radius: 8px;
  font-size: 13px;
  color: var(--text-primary);
  margin-bottom: 12px;
}

.directory-path.empty {
  color: var(--text-secondary);
  font-style: italic;
}

.chip-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  max-height: 80px;
  overflow-y: auto;
}

.chip-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: rgba(64, 158, 255, 0.1);
  border: 1px solid rgba(64, 158, 255, 0.3);
  border-radius: 6px;
  font-size: 12px;
  color: var(--primary-color);
}

.chip-item.chip-stable {
  background: rgba(128, 128, 128, 0.1);
  border-color: rgba(128, 128, 128, 0.3);
  color: var(--text-secondary);
  opacity: 0.7;
}

/* 日志区域 */
.log-section {
  flex: 1;
  min-height: 0;
}

.log-section>* {
  height: 100%;
}
</style>
