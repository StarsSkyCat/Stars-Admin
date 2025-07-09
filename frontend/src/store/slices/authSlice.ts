import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit'
import { message } from 'antd'
import { authAPI } from '../../services/auth'
import type { LoginRequest, LoginResponse, User } from '../../types'

/**
 * 认证状态接口
 */
interface AuthState {
  isAuthenticated: boolean
  user: User | null
  accessToken: string | null
  refreshToken: string | null
  loading: boolean
  error: string | null
}

/**
 * 初始状态
 */
const initialState: AuthState = {
  isAuthenticated: !!localStorage.getItem('access_token'),
  user: null,
  accessToken: localStorage.getItem('access_token'),
  refreshToken: localStorage.getItem('refresh_token'),
  loading: false,
  error: null,
}

/**
 * 异步Action：用户登录
 */
export const login = createAsyncThunk(
  'auth/login',
  async (credentials: LoginRequest) => {
    const response = await authAPI.login(credentials)
    return response
  }
)

/**
 * 异步Action：刷新token
 */
export const refreshAccessToken = createAsyncThunk(
  'auth/refreshToken',
  async (refreshToken: string) => {
    const response = await authAPI.refreshToken({ refresh_token: refreshToken })
    return response
  }
)

/**
 * 异步Action：获取用户信息
 */
export const fetchUserInfo = createAsyncThunk(
  'auth/fetchUserInfo',
  async () => {
    const response = await authAPI.getUserInfo()
    return response
  }
)

/**
 * 异步Action：用户登出
 */
export const logout = createAsyncThunk(
  'auth/logout',
  async () => {
    await authAPI.logout()
  }
)

/**
 * 认证slice
 */
const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    // 清除错误信息
    clearError: (state) => {
      state.error = null
    },
    // 直接登出（不调用API）
    logoutLocal: (state) => {
      state.isAuthenticated = false
      state.user = null
      state.accessToken = null
      state.refreshToken = null
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
    },
  },
  extraReducers: (builder) => {
    // 登录
    builder
      .addCase(login.pending, (state) => {
        state.loading = true
        state.error = null
      })
      .addCase(login.fulfilled, (state, action: PayloadAction<LoginResponse>) => {
        state.loading = false
        state.isAuthenticated = true
        state.user = action.payload.user
        state.accessToken = action.payload.access_token
        state.refreshToken = action.payload.refresh_token
        
        // 保存到localStorage
        localStorage.setItem('access_token', action.payload.access_token)
        localStorage.setItem('refresh_token', action.payload.refresh_token)
        
        message.success('登录成功')
      })
      .addCase(login.rejected, (state, action) => {
        state.loading = false
        state.error = action.error.message || '登录失败'
      })

    // 刷新token
    builder
      .addCase(refreshAccessToken.fulfilled, (state, action: PayloadAction<LoginResponse>) => {
        state.accessToken = action.payload.access_token
        state.refreshToken = action.payload.refresh_token
        
        // 更新localStorage
        localStorage.setItem('access_token', action.payload.access_token)
        localStorage.setItem('refresh_token', action.payload.refresh_token)
      })

    // 获取用户信息
    builder
      .addCase(fetchUserInfo.pending, (state) => {
        state.loading = true
      })
      .addCase(fetchUserInfo.fulfilled, (state, action: PayloadAction<User>) => {
        state.loading = false
        state.user = action.payload
      })
      .addCase(fetchUserInfo.rejected, (state, action) => {
        state.loading = false
        state.error = action.error.message || '获取用户信息失败'
      })

    // 登出
    builder
      .addCase(logout.fulfilled, (state) => {
        state.isAuthenticated = false
        state.user = null
        state.accessToken = null
        state.refreshToken = null
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        message.success('登出成功')
      })
  },
})

export const { clearError, logoutLocal } = authSlice.actions
export default authSlice.reducer