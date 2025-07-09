import axios, { AxiosInstance, AxiosResponse, AxiosError } from 'axios'
import { message } from 'antd'
import { store } from '../store'
import { logout } from '../store/slices/authSlice'

/**
 * 创建axios实例
 */
const createAxiosInstance = (): AxiosInstance => {
  const instance = axios.create({
    baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
    timeout: 10000,
    headers: {
      'Content-Type': 'application/json',
    },
  })

  // 请求拦截器
  instance.interceptors.request.use(
    (config) => {
      // 添加认证token
      const token = localStorage.getItem('access_token')
      if (token) {
        config.headers.Authorization = `Bearer ${token}`
      }
      return config
    },
    (error) => {
      return Promise.reject(error)
    }
  )

  // 响应拦截器
  instance.interceptors.response.use(
    (response: AxiosResponse) => {
      const { code, message: msg, data } = response.data

      // 业务成功
      if (code === 200) {
        return data
      }

      // 业务失败
      message.error(msg || '请求失败')
      return Promise.reject(new Error(msg || '请求失败'))
    },
    (error: AxiosError) => {
      const { response } = error

      // 处理HTTP错误状态码
      if (response) {
        const { status, data } = response

        switch (status) {
          case 401:
            // 未授权，清除token并跳转到登录页
            localStorage.removeItem('access_token')
            localStorage.removeItem('refresh_token')
            store.dispatch(logout())
            message.error('登录已过期，请重新登录')
            break
          case 403:
            message.error('没有权限访问该资源')
            break
          case 404:
            message.error('请求的资源不存在')
            break
          case 500:
            message.error('服务器内部错误')
            break
          default:
            message.error((data as any)?.message || '请求失败')
        }
      } else {
        // 网络错误或超时
        if (error.code === 'ECONNABORTED') {
          message.error('请求超时，请检查网络连接')
        } else {
          message.error('网络错误，请检查网络连接')
        }
      }

      return Promise.reject(error)
    }
  )

  return instance
}

// 创建axios实例
export const request = createAxiosInstance()

// 导出常用方法
export const api = {
  get: <T = any>(url: string, params?: any) => request.get<T>(url, { params }),
  post: <T = any>(url: string, data?: any) => request.post<T>(url, data),
  put: <T = any>(url: string, data?: any) => request.put<T>(url, data),
  delete: <T = any>(url: string, params?: any) => request.delete<T>(url, { params }),
  patch: <T = any>(url: string, data?: any) => request.patch<T>(url, data),
}

export default request