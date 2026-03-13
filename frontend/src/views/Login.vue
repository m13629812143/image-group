<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import http, { saveLogin } from '../utils/auth'

const router = useRouter()
const username = ref('')
const password = ref('')
const message = ref('')
const loading = ref(false)

async function handleLogin() {
  if (!username.value || !password.value) {
    message.value = '请输入用户名和密码'
    return
  }

  loading.value = true
  message.value = ''

  try {
    const res = await http.post('/login', {
      username: username.value,
      password: password.value,
    })
    if (res.data.code === 0) {
      saveLogin(res.data.data.token, res.data.data.username, res.data.data.role)
      router.push('/files')
    } else {
      message.value = res.data.message
    }
  } catch (err) {
    message.value = err.response?.data?.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <h2>登录</h2>
      <div v-if="message" class="message error">{{ message }}</div>
      <form @submit.prevent="handleLogin">
        <div class="form-group">
          <label>用户名</label>
          <input v-model="username" type="text" placeholder="请输入用户名" />
        </div>
        <div class="form-group">
          <label>密码</label>
          <input v-model="password" type="password" placeholder="请输入密码" />
        </div>
        <button type="submit" class="btn btn-primary btn-block" :disabled="loading">
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>
      <p class="auth-link">
        没有账号？<router-link to="/register">去注册</router-link>
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
