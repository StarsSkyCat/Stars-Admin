import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from 'antd'
import { useSelector } from 'react-redux'

import { RootState } from './store'
import LoginPage from './pages/Login'
import DashboardLayout from './components/Layout/DashboardLayout'
import Dashboard from './pages/Dashboard'
import UserList from './pages/System/UserList'
import RoleList from './pages/System/RoleList'
import MenuList from './pages/System/MenuList'
import OperationLogs from './pages/System/OperationLogs'
import NotFound from './pages/NotFound'

const { Content } = Layout

/**
 * 应用主组件
 */
function App() {
  const { isAuthenticated } = useSelector((state: RootState) => state.auth)

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Content>
        <Routes>
          {/* 登录页面 */}
          <Route 
            path="/login" 
            element={isAuthenticated ? <Navigate to="/dashboard" /> : <LoginPage />} 
          />
          
          {/* 受保护的路由 */}
          <Route 
            path="/*" 
            element={
              isAuthenticated ? (
                <DashboardLayout>
                  <Routes>
                    <Route path="/" element={<Navigate to="/dashboard" />} />
                    <Route path="/dashboard" element={<Dashboard />} />
                    
                    {/* 系统管理 */}
                    <Route path="/system/users" element={<UserList />} />
                    <Route path="/system/roles" element={<RoleList />} />
                    <Route path="/system/menus" element={<MenuList />} />
                    <Route path="/system/logs" element={<OperationLogs />} />
                    
                    {/* 404页面 */}
                    <Route path="*" element={<NotFound />} />
                  </Routes>
                </DashboardLayout>
              ) : (
                <Navigate to="/login" />
              )
            } 
          />
        </Routes>
      </Content>
    </Layout>
  )
}

export default App