import { createRouter, createWebHashHistory } from 'vue-router'
import { isLoggedIn, isAdmin } from '../utils/auth'

import Login from '../views/Login.vue'
import Register from '../views/Register.vue'
import Files from '../views/Files.vue'
import Admin from '../views/Admin.vue'

const routes = [
  { path: '/', redirect: '/files' },
  { path: '/login', component: Login },
  { path: '/register', component: Register },
  { path: '/files', component: Files, meta: { requiresAuth: true } },
  { path: '/admin', component: Admin, meta: { requiresAuth: true, requiresAdmin: true } },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

// 路由守卫：未登录跳转到登录页，非管理员不能进管理页
router.beforeEach((to, from, next) => {
  if (to.meta.requiresAuth && !isLoggedIn()) {
    next('/login')
  } else if (to.meta.requiresAdmin && !isAdmin()) {
    next('/files')
  } else {
    next()
  }
})

export default router
