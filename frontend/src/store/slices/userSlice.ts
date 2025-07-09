import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit'
import type { User, PageRequest, PageResponse } from '../../types'

/**
 * 用户状态接口
 */
interface UserState {
  list: User[]
  total: number
  current: User | null
  loading: boolean
  error: string | null
}

/**
 * 初始状态
 */
const initialState: UserState = {
  list: [],
  total: 0,
  current: null,
  loading: false,
  error: null,
}

/**
 * 异步Action：获取用户列表
 */
export const fetchUserList = createAsyncThunk(
  'user/fetchUserList',
  async (params: PageRequest) => {
    // 这里应该调用用户API，暂时返回模拟数据
    const response: PageResponse<User> = {
      list: [],
      total: 0,
      page: params.page || 1,
      page_size: params.page_size || 10,
    }
    return response
  }
)

/**
 * 用户slice
 */
const userSlice = createSlice({
  name: 'user',
  initialState,
  reducers: {
    // 清除错误信息
    clearError: (state) => {
      state.error = null
    },
    // 设置当前用户
    setCurrent: (state, action: PayloadAction<User | null>) => {
      state.current = action.payload
    },
  },
  extraReducers: (builder) => {
    // 获取用户列表
    builder
      .addCase(fetchUserList.pending, (state) => {
        state.loading = true
        state.error = null
      })
      .addCase(fetchUserList.fulfilled, (state, action: PayloadAction<PageResponse<User>>) => {
        state.loading = false
        state.list = action.payload.list
        state.total = action.payload.total
      })
      .addCase(fetchUserList.rejected, (state, action) => {
        state.loading = false
        state.error = action.error.message || '获取用户列表失败'
      })
  },
})

export const { clearError, setCurrent } = userSlice.actions
export default userSlice.reducer