<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import http from '../utils/auth'

const router = useRouter()
const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const message = ref('')
const messageType = ref('error')
const loading = ref(false)

async function handleRegister() {
  if (!username.value || !password.value) {
    message.value = '请输入用户名和密码'
    messageType.value = 'error'
    return
  }

  if (password.value !== confirmPassword.value) {
    message.value = '两次密码不一致'
    messageType.value = 'error'
    return
  }

  if (password.value.length < 6) {
    message.value = '密码至少 6 个字符'
    messageType.value = 'error'
    return
  }

  loading.value = true
  message.value = ''

  try {
    const res = await http.post('/register', {
      username: username.value,
      password: password.value,
    })
    if (res.data.code === 0) {
      message.value = '注册成功，即将跳转到登录页...'
      messageType.value = 'success'
      setTimeout(() => router.push('/login'), 1500)
    } else {
      message.value = res.data.message
      messageType.value = 'error'
    }
  } catch (err) {
    message.value = err.response?.data?.message || '注册失败'
    messageType.value = 'error'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <h2>注册</h2>
      <div v-if="message" :class="['message', messageType]">{{ message }}</div>
      <form @submit.prevent="handleRegister">
        <div class="form-group">
          <label>用户名</label>
          <input v-model="username" type="text" placeholder="2-20 个字符" />
        </div>
        <div class="form-group">
          <label>密码</label>
          <input v-model="password" type="password" placeholder="至少 6 个字符" />
        </div>
        <div class="form-group">
          <label>确认密码</label>
          <input v-model="confirmPassword" type="password" placeholder="再次输入密码" />
        </div>
        <button type="submit" class="btn btn-primary btn-block" :disabled="loading">
          {{ loading ? '注册中...' : '注册' }}
        </button>
      </form>
      <p class="auth-link">
        已有账号？<router-link to="/login">去登录</router-link>
      </p>
    </div>
  </div>
</template>

<style scoped>
.auth-page {
  display: flex;
  justify-content: center;
  padding-top: 60px;
}

.auth-card {
  background: #fff;
  border-radius: 8px;
  padding: 32px;
  width: 400px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.auth-card h2 {
  text-align: center;
  margin-bottom: 24px;
  color: #2c3e50;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 6px;
  font-size: 14px;
  color: #555;
}

.form-group input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
}

.form-group input:focus {
  border-color: #3498db;
}

.btn-block {
  width: 100%;
  margin-top: 8px;
}

.auth-link {
  text-align: center;
  margin-top: 16px;
  font-size: 14px;
  color: #7f8c8d;
}

.auth-link a {
  color: #3498db;
  text-decoration: none;
}
</style>
