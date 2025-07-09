import { api } from '../utils/request'
import type { 
  LoginRequest, 
  LoginResponse, 
  RefreshTokenRequest, 
  UpdatePasswordRequest,
  User 
} from '../types'

/**
 * 认证API服务
 */
export const authAPI = {
  // 用户登录
  login: (data: LoginRequest) => api.post<LoginResponse>('/auth/login', data),
  
  // 刷新token
  refreshToken: (data: RefreshTokenRequest) => api.post<LoginResponse>('/auth/refresh', data),
  
  // 用户登出
  logout: () => api.post('/auth/logout'),
  
  // 获取用户信息
  getUserInfo: () => api.get<User>('/auth/user'),
  
  // 更新密码
  updatePassword: (data: UpdatePasswordRequest) => api.put('/auth/password', data),
}