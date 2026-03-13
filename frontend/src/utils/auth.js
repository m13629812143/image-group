// Token 管理工具
import axios from 'axios'

const API = import.meta.env.DEV ? 'http://localhost:8080/api' : '/api'

// 创建带 Token 的 axios 实例
const http = axios.create({ baseURL: API })

// 请求拦截器：自动在请求头加上 Token
http.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器：Token 过期时自动跳转到登录页
http.interceptors.response.use(
  response => response,
  error => {
    if (error.response && error.response.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('username')
      localStorage.removeItem('role')
      window.location.hash = '#/login'
    }
    return Promise.reject(error)
  }
)

// 保存登录信息
export function saveLogin(token, username, role) {
  localStorage.setItem('token', token)
  localStorage.setItem('username', username)
  localStorage.setItem('role', role)
}

// 清除登录信息
export function logout() {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  localStorage.removeItem('role')
}

// 获取当前用户信息
export function getUser() {
  return {
    token: localStorage.getItem('token'),
    username: localStorage.getItem('username'),
    role: localStorage.getItem('role'),
  }
}

// 是否已登录
export function isLoggedIn() {
  return !!localStorage.getItem('token')
}

// 是否是管理员
export function isAdmin() {
  return localStorage.getItem('role') === 'admin'
}

export default http
