<script setup>
import { ref, onMounted } from 'vue'
import http from '../utils/auth'

const activeTab = ref('stats')
const stats = ref(null)
const users = ref([])
const files = ref([])
const message = ref('')
const messageType = ref('')

// 获取统计数据
async function fetchStats() {
  try {
    const res = await http.get('/admin/stats')
    if (res.data.code === 0) {
      stats.value = res.data.data
    }
  } catch (err) {
    console.error('获取统计失败:', err)
  }
}

// 获取用户列表
async function fetchUsers() {
  try {
    const res = await http.get('/admin/users')
    if (res.data.code === 0) {
      users.value = res.data.data
    }
  } catch (err) {
    console.error('获取用户列表失败:', err)
  }
}

// 获取所有文件
async function fetchFiles() {
  try {
    const res = await http.get('/admin/files')
    if (res.data.code === 0) {
      files.value = res.data.data
    }
  } catch (err) {
    console.error('获取文件列表失败:', err)
  }
}

// 删除用户
async function deleteUser(id, username) {
  if (!confirm(`确定删除用户 "${username}" 及其所有文件吗？`)) return

  try {
    const res = await http.delete(`/admin/users/${id}`)
    if (res.data.code === 0) {
      showMessage('用户已删除', 'success')
      fetchUsers()
      fetchStats()
    }
  } catch (err) {
    showMessage(err.response?.data?.message || '删除失败', 'error')
  }
}

// 删除文件
async function deleteFile(id, filename) {
  if (!confirm(`确定删除文件 "${filename}" 吗？`)) return

  try {
    const res = await http.delete(`/admin/files/${id}`)
    if (res.data.code === 0) {
      showMessage('文件已删除', 'success')
      fetchFiles()
      fetchStats()
    }
  } catch (err) {
    showMessage(err.response?.data?.message || '删除失败', 'error')
  }
}

// 切换标签页
function switchTab(tab) {
  activeTab.value = tab
  if (tab === 'stats') fetchStats()
  if (tab === 'users') fetchUsers()
  if (tab === 'files') fetchFiles()
}

// 工具函数
function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
}

function formatTime(time) {
  return new Date(time).toLocaleString('zh-CN')
}

function showMessage(msg, type) {
  message.value = msg
  messageType.value = type
  setTimeout(() => { message.value = '' }, 3000)
}

onMounted(() => {
  fetchStats()
})
</script>

<template>
  <div>
    <div v-if="message" :class="['message', messageType]">{{ message }}</div>

    <!-- 标签切换 -->
    <div class="tabs">
      <button :class="['tab', activeTab === 'stats' && 'active']" @click="switchTab('stats')">数据统计</button>
      <button :class="['tab', activeTab === 'users' && 'active']" @click="switchTab('users')">用户管理</button>
      <button :class="['tab', activeTab === 'files' && 'active']" @click="switchTab('files')">文件管理</button>
    </div>

    <!-- 数据统计 -->
    <div v-if="activeTab === 'stats'" class="card">
      <h2>数据统计</h2>
      <div v-if="stats" class="stats-grid">
        <div class="stat-item">
          <div class="stat-value">{{ stats.user_count }}</div>
          <div class="stat-label">用户总数</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">{{ stats.file_count }}</div>
          <div class="stat-label">文件总数</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">{{ formatSize(stats.total_size) }}</div>
          <div class="stat-label">总存储量</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">{{ stats.upload_count || 0 }}</div>
          <div class="stat-label">累计上传次数</div>
        </div>
      </div>
      <div v-if="stats" class="service-status">
        <h3>服务状态</h3>
        <p>
          <span :class="stats.mongodb === '正常' ? 'dot green' : 'dot red'"></span>
          MongoDB: {{ stats.mongodb }}
        </p>
        <p>
          <span :class="stats.redis === '正常' ? 'dot green' : 'dot red'"></span>
          Redis: {{ stats.redis }}
        </p>
      </div>
    </div>

    <!-- 用户管理 -->
    <div v-if="activeTab === 'users'" class="card">
      <h2>用户管理 ({{ users.length }})</h2>
      <div v-if="users.length === 0" class="empty">暂无用户</div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>用户名</th>
            <th>角色</th>
            <th>注册时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id">
            <td>{{ user.username }}</td>
            <td>
              <span :class="['role-badge', user.role]">
                {{ user.role === 'admin' ? '管理员' : '普通用户' }}
              </span>
            </td>
            <td>{{ formatTime(user.created_at) }}</td>
            <td>
              <button
                v-if="user.role !== 'admin'"
                @click="deleteUser(user.id, user.username)"
                class="btn btn-danger btn-sm"
              >删除</button>
              <span v-else class="text-muted">-</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 文件管理 -->
    <div v-if="activeTab === 'files'" class="card">
      <h2>文件管理 ({{ files.length }})</h2>
      <div v-if="files.length === 0" class="empty">暂无文件</div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>文件名</th>
            <th>上传者</th>
            <th>大小</th>
            <th>上传时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="file in files" :key="file.id">
            <td>{{ file.filename }}</td>
            <td>{{ file.uploader }}</td>
            <td>{{ formatSize(file.size) }}</td>
            <td>{{ formatTime(file.upload_time) }}</td>
            <td>
              <button @click="deleteFile(file.id, file.filename)" class="btn btn-danger btn-sm">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
/* 标签切换 */
.tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 20px;
  background: #fff;
  padding: 4px;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}

.tab {
  flex: 1;
  padding: 10px;
  border: none;
  background: none;
  cursor: pointer;
  font-size: 14px;
  border-radius: 6px;
  color: #555;
  transition: all 0.2s;
}

.tab.active {
  background: #3498db;
  color: #fff;
  font-weight: 600;
}

.tab:hover:not(.active) {
  background: #ecf0f1;
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-item {
  text-align: center;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #2c3e50;
}

.stat-label {
  font-size: 13px;
  color: #7f8c8d;
  margin-top: 4px;
}

/* 服务状态 */
.service-status {
  padding-top: 16px;
  border-top: 1px solid #ecf0f1;
}

.service-status h3 {
  font-size: 15px;
  color: #2c3e50;
  margin-bottom: 8px;
}

.service-status p {
  font-size: 14px;
  color: #555;
  margin-bottom: 4px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}

.dot.green { background: #27ae60; }
.dot.red { background: #e74c3c; }

/* 角色标签 */
.role-badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.role-badge.admin {
  background: #fff3cd;
  color: #856404;
}

.role-badge.user {
  background: #d1ecf1;
  color: #0c5460;
}

.text-muted {
  color: #bdc3c7;
}
</style>
