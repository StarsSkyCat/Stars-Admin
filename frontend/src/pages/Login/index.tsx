import React from 'react'
import { Form, Input, Button, Card, message } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import { AppDispatch, RootState } from '../store'
import { login, clearError } from '../store/slices/authSlice'
import type { LoginRequest } from '../types'
import './Login.css'

/**
 * 登录页面组件
 */
const LoginPage: React.FC = () => {
  const [form] = Form.useForm()
  const navigate = useNavigate()
  const dispatch = useDispatch<AppDispatch>()
  const { loading, error } = useSelector((state: RootState) => state.auth)

  /**
   * 处理登录表单提交
   */
  const handleSubmit = async (values: LoginRequest) => {
    try {
      await dispatch(login(values)).unwrap()
      message.success('登录成功')
      navigate('/dashboard')
    } catch (error) {
      console.error('登录失败:', error)
    }
  }

  // 清除错误信息
  React.useEffect(() => {
    return () => {
      dispatch(clearError())
    }
  }, [dispatch])

  return (
    <div className="login-container">
      <Card className="login-card" title="Stars Admin">
        <Form
          form={form}
          name="login"
          onFinish={handleSubmit}
          autoComplete="off"
          size="large"
        >
          <Form.Item
            name="username"
            rules={[
              { required: true, message: '请输入用户名' },
              { min: 3, message: '用户名至少3个字符' },
            ]}
          >
            <Input
              prefix={<UserOutlined />}
              placeholder="用户名"
              autoComplete="username"
            />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 6, message: '密码至少6个字符' },
            ]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="密码"
              autoComplete="current-password"
            />
          </Form.Item>

          {error && (
            <div className="error-message">
              {error}
            </div>
          )}

          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              block
            >
              登录
            </Button>
          </Form.Item>
        </Form>
        
        <div className="login-tips">
          <p>默认账号：admin</p>
          <p>默认密码：admin123</p>
        </div>
      </Card>
    </div>
  )
}

export default LoginPage