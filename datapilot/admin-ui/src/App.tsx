import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import { theme } from './theme'
import Layout from './components/Layout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Files from './pages/Files'
import Scheduler from './pages/Scheduler'

function isTokenValid(): boolean {
  const token = localStorage.getItem('datapilot_token')
  if (!token) return false
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return typeof payload.exp === 'number' && payload.exp > Date.now() / 1000
  } catch {
    return false
  }
}

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  if (!isTokenValid()) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

export default function App() {
  return (
    <ConfigProvider theme={theme}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <Layout />
              </ProtectedRoute>
            }
          >
            <Route index element={<Dashboard />} />
            <Route path="files" element={<Files />} />
            <Route path="scheduler" element={<Scheduler />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  )
}
