// API响应类型
export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

// 分页响应类型
export interface PageResponse<T = any> {
  list: T[]
  total: number
  page: number
  page_size: number
}

// 用户类型
export interface User {
  id: number
  username: string
  email: string
  phone?: string
  nickname?: string
  avatar?: string
  status: number
  last_login_at?: string
  created_at: string
  updated_at: string
  roles?: Role[]
}

// 角色类型
export interface Role {
  id: number
  name: string
  code: string
  description?: string
  status: number
  created_at: string
  updated_at: string
  users?: User[]
  menus?: Menu[]
}

// 菜单类型
export interface Menu {
  id: number
  parent_id: number
  name: string
  path?: string
  component?: string
  icon?: string
  sort: number
  type: number // 1:菜单 2:按钮
  status: number
  created_at: string
  updated_at: string
  children?: Menu[]
  roles?: Role[]
}

// 权限类型
export interface Permission {
  id: number
  name: string
  code: string
  description?: string
  created_at: string
  updated_at: string
}

// 操作日志类型
export interface OperationLog {
  id: number
  user_id: number
  username: string
  method: string
  path: string
  ip: string
  user_agent: string
  status: number
  latency: number
  request: string
  response: string
  created_at: string
}

// 登录请求类型
export interface LoginRequest {
  username: string
  password: string
}

// 登录响应类型
export interface LoginResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: User
}

// 刷新令牌请求类型
export interface RefreshTokenRequest {
  refresh_token: string
}

// 更新密码请求类型
export interface UpdatePasswordRequest {
  old_password: string
  new_password: string
}

// 分页请求参数
export interface PageRequest {
  page?: number
  page_size?: number
  keyword?: string
  status?: number
}

// 表单模式
export type FormMode = 'create' | 'edit' | 'view'

// 表格操作类型
export type TableAction = 'create' | 'edit' | 'delete' | 'view'