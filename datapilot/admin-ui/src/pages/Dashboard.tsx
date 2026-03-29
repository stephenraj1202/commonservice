import { useEffect, useState } from 'react'
import { Row, Col, Spin } from 'antd'
import {
  FileOutlined,
  DatabaseOutlined,
  ScheduleOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons'
import StatCard from '../components/StatCard'
import { listFiles } from '../api/files'
import { listJobs } from '../api/scheduler'
import { tealColor, amberColor } from '../theme'

interface DashboardStats {
  totalFiles: number
  storageMB: number
  totalJobs: number
  activeJobs: number
}

export default function Dashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function fetchStats() {
      try {
        const [filesRes, jobsRes, activeJobsRes] = await Promise.all([
          listFiles(1, 1),
          listJobs(1, 1),
          listJobs(1, 1, 'active'),
        ])

        const storageMB =
          filesRes.data.reduce((sum, f) => sum + f.size_bytes, 0) / (1024 * 1024)

        setStats({
          totalFiles: filesRes.total,
          storageMB: Math.round(storageMB * 100) / 100,
          totalJobs: jobsRes.total,
          activeJobs: activeJobsRes.total,
        })
      } finally {
        setLoading(false)
      }
    }

    fetchStats()
  }, [])

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', paddingTop: 80 }}>
        <Spin size="large" />
      </div>
    )
  }

  return (
    <Row gutter={[24, 24]}>
      <Col xs={24} sm={12} xl={6}>
        <StatCard
          title="Total Files"
          value={stats?.totalFiles ?? 0}
          icon={<FileOutlined />}
          color={tealColor}
        />
      </Col>
      <Col xs={24} sm={12} xl={6}>
        <StatCard
          title="Storage Used (MB)"
          value={stats?.storageMB ?? 0}
          icon={<DatabaseOutlined />}
          color={tealColor}
        />
      </Col>
      <Col xs={24} sm={12} xl={6}>
        <StatCard
          title="Total Jobs"
          value={stats?.totalJobs ?? 0}
          icon={<ScheduleOutlined />}
          color={amberColor}
        />
      </Col>
      <Col xs={24} sm={12} xl={6}>
        <StatCard
          title="Active Jobs"
          value={stats?.activeJobs ?? 0}
          icon={<CheckCircleOutlined />}
          color={amberColor}
        />
      </Col>
    </Row>
  )
}
