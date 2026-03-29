import { Layout as AntLayout, Menu, Button, theme as antTheme } from 'antd'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import {
  DashboardOutlined,
  FileOutlined,
  ScheduleOutlined,
  LogoutOutlined,
  FileTextOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '../store/auth'

const { Sider, Header, Content } = AntLayout

const menuItems = [
  { key: '/', icon: <DashboardOutlined />, label: 'Dashboard' },
  { key: '/files', icon: <FileOutlined />, label: 'Files' },
  { key: '/scheduler', icon: <ScheduleOutlined />, label: 'Scheduler' },
  { key: '/documentation', icon: <FileTextOutlined />, label: 'Documentation' },
]

export default function Layout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { token } = antTheme.useToken()

  const selectedKey = menuItems
    .slice()
    .reverse()
    .find((item) => location.pathname === item.key || location.pathname.startsWith(item.key + '/'))
    ?.key ?? '/'

  function handleMenuClick({ key }: { key: string }) {
    navigate(key)
  }

  function handleLogout() {
    useAuthStore.getState().logout()
  }

  return (
    <AntLayout style={{ minHeight: '100vh' }}>
      <Sider theme="dark" collapsible>
        <div
          style={{
            height: 32,
            margin: 16,
            color: '#fff',
            fontWeight: 700,
            fontSize: 18,
            letterSpacing: 1,
          }}
        >
          DataPilot
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={handleMenuClick}
        />
      </Sider>
      <AntLayout>
        <Header
          style={{
            background: token.colorBgContainer,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '0 24px',
          }}
        >
          <span style={{ fontWeight: 700, fontSize: 20 }}>DataPilot</span>
          <Button
            icon={<LogoutOutlined />}
            onClick={handleLogout}
            type="text"
          >
            Logout
          </Button>
        </Header>
        <Content style={{ margin: 24 }}>
          <Outlet />
        </Content>
      </AntLayout>
    </AntLayout>
  )
}
