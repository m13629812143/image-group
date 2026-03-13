<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { getUser, isLoggedIn, isAdmin, logout } from './utils/auth'

const router = useRouter()

const loggedIn = computed(() => isLoggedIn())
const user = computed(() => getUser())
const admin = computed(() => isAdmin())

function handleLogout() {
  logout()
  // 全页面刷新，确保导航栏状态完全更新
  window.location.href = '/login'
}
</script>

<template>
  <div class="app">
    <!-- 顶部导航栏 -->
    <header class="header">
      <div class="header-left">
        <h1 @click="router.push('/files')" class="logo">文件管理系统</h1>
      </div>
      <nav class="header-right" v-if="loggedIn">
        <router-link to="/files" class="nav-link">我的文件</router-link>
        <router-link to="/admin" class="nav-link" v-if="admin">管理后台</router-link>
        <span class="user-info">{{ user.username }}</span>
        <button @click="handleLogout" class="btn btn-logout">退出</button>
      </nav>
    </header>

    <!-- 页面内容 -->
    <main class="main">
      <router-view />
    </main>
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
  max-width: 1000px;
  margin: 0 auto;
  padding: 0 20px;
}

/* 顶部导航 */
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 0;
  border-bottom: 1px solid #e8e8e8;
  margin-bottom: 24px;
}

.header-left .logo {
  font-size: 22px;
  color: #2c3e50;
  cursor: pointer;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.nav-link {
  color: #555;
  text-decoration: none;
  font-size: 14px;
  padding: 6px 12px;
  border-radius: 6px;
  transition: background 0.2s;
}

.nav-link:hover {
  background: #e8e8e8;
}

.nav-link.router-link-active {
  color: #3498db;
  font-weight: 600;
}

.user-info {
  font-size: 14px;
  color: #7f8c8d;
}

/* 通用按钮 */
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

.btn-primary { background: #3498db; color: #fff; }
.btn-primary:hover:not(:disabled) { background: #2980b9; }

.btn-danger { background: #e74c3c; color: #fff; }
.btn-danger:hover:not(:disabled) { background: #c0392b; }

.btn-success { background: #27ae60; color: #fff; }
.btn-success:hover:not(:disabled) { background: #219a52; }

.btn-logout { background: #95a5a6; color: #fff; padding: 6px 14px; font-size: 13px; }
.btn-logout:hover { background: #7f8c8d; }

.btn-sm { padding: 4px 12px; font-size: 12px; }

/* 卡片 */
.card {
  background: #fff;
  border-radius: 8px;
  padding: 24px;
  margin-bottom: 20px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}

.card h2 {
  font-size: 18px;
  margin-bottom: 16px;
  color: #2c3e50;
}

/* 消息提示 */
.message {
  padding: 12px 16px;
  border-radius: 8px;
  margin-bottom: 20px;
  text-align: center;
  font-size: 14px;
}

.message.success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
.message.error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }

/* 表格 */
.table {
  width: 100%;
  border-collapse: collapse;
}

.table th, .table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid #ecf0f1;
}

.table th {
  background: #f8f9fa;
  color: #7f8c8d;
  font-size: 13px;
  font-weight: 600;
}

.table td { font-size: 14px; }
.table tr:hover { background: #f8f9fa; }

.empty {
  text-align: center;
  color: #bdc3c7;
  padding: 40px;
  font-size: 16px;
}

.main {
  padding-bottom: 40px;
}
</style>
