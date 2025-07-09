import React from 'react'
import { Card, Row, Col, Statistic, Table, Tag } from 'antd'
import { 
  UserOutlined, 
  TeamOutlined, 
  MenuOutlined, 
  FileTextOutlined 
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { OperationLog } from '../types'

/**
 * 仪表板页面组件
 */
const Dashboard: React.FC = () => {
  // 模拟数据
  const recentLogs: OperationLog[] = [
    {
      id: 1,
      user_id: 1,
      username: 'admin',
      method: 'GET',
      path: '/api/v1/users',
      ip: '127.0.0.1',
      user_agent: 'Mozilla/5.0',
      status: 200,
      latency: 45,
      request: '',
      response: '',
      created_at: '2024-01-01 10:00:00',
    },
    {
      id: 2,
      user_id: 1,
      username: 'admin',
      method: 'POST',
      path: '/api/v1/users',
      ip: '127.0.0.1',
      user_agent: 'Mozilla/5.0',
      status: 201,
      latency: 123,
      request: '',
      response: '',
      created_at: '2024-01-01 09:30:00',
    },
  ]

  // 表格列定义
  const columns: ColumnsType<OperationLog> = [
    {
      title: '用户',
      dataIndex: 'username',
      key: 'username',
    },
    {
      title: '方法',
      dataIndex: 'method',
      key: 'method',
      render: (method: string) => (
        <Tag color={getMethodColor(method)}>{method}</Tag>
      ),
    },
    {
      title: '路径',
      dataIndex: 'path',
      key: 'path',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: number) => (
        <Tag color={status === 200 ? 'success' : 'error'}>{status}</Tag>
      ),
    },
    {
      title: '耗时',
      dataIndex: 'latency',
      key: 'latency',
      render: (latency: number) => `${latency}ms`,
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
    },
  ]

  /**
   * 获取HTTP方法对应的颜色
   */
  function getMethodColor(method: string): string {
    switch (method) {
      case 'GET':
        return 'blue'
      case 'POST':
        return 'green'
      case 'PUT':
        return 'orange'
      case 'DELETE':
        return 'red'
      default:
        return 'default'
    }
  }

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="用户总数"
              value={1234}
              prefix={<UserOutlined />}
              valueStyle={{ color: '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="角色数量"
              value={12}
              prefix={<TeamOutlined />}
              valueStyle={{ color: '#cf1322' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="菜单数量"
              value={28}
              prefix={<MenuOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="今日访问"
              value={567}
              prefix={<FileTextOutlined />}
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
      </Row>

      <Card title="最近操作记录">
        <Table
          columns={columns}
          dataSource={recentLogs}
          rowKey="id"
          size="small"
          pagination={false}
        />
      </Card>
    </div>
  )
}

export default Dashboard