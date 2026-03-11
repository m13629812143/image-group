<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

// 开发模式直连后端，生产模式通过 nginx 代理
const API = import.meta.env.DEV ? 'http://localhost:8080/api' : '/api'

// 响应式数据
const files = ref([])
const selectedFile = ref(null)
const uploading = ref(false)
const message = ref('')
const messageType = ref('')
const health = ref(null)

// 选择文件
function onFileChange(e) {
  selectedFile.value = e.target.files[0]
  message.value = ''
}

// 上传文件
async function upload() {
  if (!selectedFile.value) {
    showMessage('请先选择文件', 'error')
    return
  }

  uploading.value = true
  const formData = new FormData()
  formData.append('file', selectedFile.value)

  try {
    const res = await axios.post(`${API}/upload`, formData)
    if (res.data.code === 0) {
      showMessage(`上传成功: ${res.data.data.filename}`, 'success')
      selectedFile.value = null
      // 清空文件选择框
      document.getElementById('fileInput').value = ''
      // 刷新文件列表
      fetchFiles()
    } else {
      showMessage(res.data.message, 'error')
    }
  } catch (err) {
    showMessage('上传失败: ' + (err.response?.data?.message || err.message), 'error')
  } finally {
    uploading.value = false
  }
}

// 获取文件列表
async function fetchFiles() {
  try {
    const res = await axios.get(`${API}/files`)
    if (res.data.code === 0) {
      files.value = res.data.data
    }
  } catch (err) {
    console.error('获取文件列表失败:', err)
  }
}

// 删除文件
async function deleteFile(filename) {
  if (!confirm(`确定要删除 "${filename}" 吗?`)) return

  try {
    const res = await axios.delete(`${API}/files/${encodeURIComponent(filename)}`)
    if (res.data.code === 0) {
      showMessage('删除成功', 'success')
      fetchFiles()
    }
  } catch (err) {
    showMessage('删除失败', 'error')
  }
}

// 健康检查
async function checkHealth() {
  try {
    const res = await axios.get(`${API}/health`)
    health.value = res.data.data
  } catch (err) {
    health.value = { mongodb: '无法连接', redis: '无法连接' }
  }
}

// 格式化文件大小
function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

// 格式化时间
function formatTime(time) {
  return new Date(time).toLocaleString('zh-CN')
}

// 显示消息提示
function showMessage(msg, type) {
  message.value = msg
  messageType.value = type
  setTimeout(() => { message.value = '' }, 3000)
}

// 页面加载时获取数据
onMounted(() => {
  fetchFiles()
  checkHealth()
})
</script>

<template>
  <div class="app">
    <header class="header">
      <h1>Image Group - 文件上传系统</h1>
      <p class="subtitle">Vue + Gin + MongoDB + Redis</p>
    </header>

    <!-- 服务状态 -->
    <div class="status-bar" v-if="health">
      <span class="status-item">
        <span :class="health.mongodb === '正常' ? 'dot green' : 'dot red'"></span>
        MongoDB: {{ health.mongodb }}
      </span>
      <span class="status-item">
        <span :class="health.redis === '正常' ? 'dot green' : 'dot red'"></span>
        Redis: {{ health.redis }}
      </span>
      <span class="status-item" v-if="health.upload_count">
        总上传次数: {{ health.upload_count }}
      </span>
    </div>

    <!-- 消息提示 -->
    <div v-if="message" :class="['message', messageType]">
      {{ message }}
    </div>

    <!-- 上传区域 -->
    <div class="upload-section">
      <h2>上传文件</h2>
      <div class="upload-box">
        <input id="fileInput" type="file" @change="onFileChange" />
        <div class="file-info" v-if="selectedFile">
          已选择: {{ selectedFile.name }} ({{ formatSize(selectedFile.size) }})
        </div>
        <button @click="upload" :disabled="uploading || !selectedFile" class="btn btn-upload">
          {{ uploading ? '上传中...' : '上传文件' }}
        </button>
      </div>
    </div>

    <!-- 文件列表 -->
    <div class="file-section">
      <h2>已上传的文件 ({{ files.length }})</h2>
      <div v-if="files.length === 0" class="empty">暂无文件</div>
      <table v-else class="file-table">
        <thead>
          <tr>
            <th>文件名</th>
            <th>大小</th>
            <th>上传时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="file in files" :key="file.id">
            <td>{{ file.filename }}</td>
            <td>{{ formatSize(file.size) }}</td>
            <td>{{ formatTime(file.upload_time) }}</td>
            <td>
              <button @click="deleteFile(file.filename)" class="btn btn-delete">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  background: #f5f7fa;
  color: #333;
}

.app {
  max-width: 900px;
  margin: 0 auto;
  padding: 20px;
}

.header {
  text-align: center;
  padding: 30px 0;
}

.header h1 {
  font-size: 28px;
  color: #2c3e50;
}

.subtitle {
  color: #7f8c8d;
  margin-top: 8px;
  font-size: 14px;
}

/* 服务状态栏 */
.status-bar {
  display: flex;
  gap: 20px;
  justify-content: center;
  padding: 12px;
  background: #fff;
  border-radius: 8px;
  margin-bottom: 20px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  font-size: 14px;
}

.status-item {
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

/* 消息提示 */
.message {
  padding: 12px 16px;
  border-radius: 8px;
  margin-bottom: 20px;
  text-align: center;
  font-size: 14px;
}

.message.success {
  background: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.message.error {
  background: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

/* 上传区域 */
.upload-section {
  background: #fff;
  border-radius: 8px;
  padding: 24px;
  margin-bottom: 20px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}

.upload-section h2 {
  font-size: 18px;
  margin-bottom: 16px;
  color: #2c3e50;
}

.upload-box {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.file-info {
  color: #7f8c8d;
  font-size: 14px;
}

/* 按钮 */
.btn {
  padding: 8px 20px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  transition: opacity 0.2s;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-upload {
  background: #3498db;
  color: #fff;
}

.btn-upload:hover:not(:disabled) {
  background: #2980b9;
}

.btn-delete {
  background: #e74c3c;
  color: #fff;
  padding: 4px 12px;
  font-size: 12px;
}

.btn-delete:hover {
  background: #c0392b;
}

/* 文件列表 */
.file-section {
  background: #fff;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}

.file-section h2 {
  font-size: 18px;
  margin-bottom: 16px;
  color: #2c3e50;
}

.empty {
  text-align: center;
  color: #bdc3c7;
  padding: 40px;
  font-size: 16px;
}

.file-table {
  width: 100%;
  border-collapse: collapse;
}

.file-table th,
.file-table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid #ecf0f1;
}

.file-table th {
  background: #f8f9fa;
  color: #7f8c8d;
  font-size: 13px;
  font-weight: 600;
}

.file-table td {
  font-size: 14px;
}

.file-table tr:hover {
  background: #f8f9fa;
}
</style>
