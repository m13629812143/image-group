<script setup>
import { ref, onMounted } from 'vue'
import http, { getUser } from '../utils/auth'

const files = ref([])
const selectedFile = ref(null)
const uploading = ref(false)
const message = ref('')
const messageType = ref('')
const user = getUser()

// 获取文件列表（只返回当前用户的文件）
async function fetchFiles() {
  try {
    const res = await http.get('/files')
    if (res.data.code === 0) {
      files.value = res.data.data
    }
  } catch (err) {
    console.error('获取文件列表失败:', err)
  }
}

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
    const res = await http.post('/upload', formData)
    if (res.data.code === 0) {
      showMessage(`上传成功: ${res.data.data.filename}`, 'success')
      selectedFile.value = null
      document.getElementById('fileInput').value = ''
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

// 下载文件
function downloadFile(id, filename) {
  const token = localStorage.getItem('token')
  // 使用 fetch 带上 Token 下载
  const baseURL = import.meta.env.DEV ? 'http://localhost:8080/api' : '/api'
  fetch(`${baseURL}/download/${id}`, {
    headers: { 'Authorization': `Bearer ${token}` }
  })
  .then(res => res.blob())
  .then(blob => {
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  })
  .catch(() => showMessage('下载失败', 'error'))
}

// 删除文件
async function deleteFile(id, filename) {
  if (!confirm(`确定要删除 "${filename}" 吗?`)) return

  try {
    const res = await http.delete(`/files/${id}`)
    if (res.data.code === 0) {
      showMessage('删除成功', 'success')
      fetchFiles()
    }
  } catch (err) {
    showMessage(err.response?.data?.message || '删除失败', 'error')
  }
}

// 工具函数
function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
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
  fetchFiles()
})
</script>

<template>
  <div>
    <!-- 消息提示 -->
    <div v-if="message" :class="['message', messageType]">{{ message }}</div>

    <!-- 上传区域 -->
    <div class="card">
      <h2>上传文件</h2>
      <div class="upload-box">
        <input id="fileInput" type="file" @change="onFileChange" />
        <div class="file-info" v-if="selectedFile">
          已选择: {{ selectedFile.name }} ({{ formatSize(selectedFile.size) }})
        </div>
        <button @click="upload" :disabled="uploading || !selectedFile" class="btn btn-primary">
          {{ uploading ? '上传中...' : '上传文件' }}
        </button>
      </div>
    </div>

    <!-- 文件列表 -->
    <div class="card">
      <h2>我的文件 ({{ files.length }})</h2>
      <div v-if="files.length === 0" class="empty">暂无文件，上传一个试试吧</div>
      <table v-else class="table">
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
              <button @click="downloadFile(file.id, file.filename)" class="btn btn-success btn-sm" style="margin-right:6px">下载</button>
              <button @click="deleteFile(file.id, file.filename)" class="btn btn-danger btn-sm">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
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
</style>
