import { useEffect, useState } from 'react'
import { Drawer, Table, Badge } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { getJobLogs, type JobExecutionLog } from '../api/scheduler'

interface LogDrawerProps {
  open: boolean
  jobId: number | null
  onClose: () => void
}

export default function LogDrawer({ open, jobId, onClose }: LogDrawerProps) {
  const [logs, setLogs] = useState<JobExecutionLog[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open || jobId === null) return
    setLoading(true)
    getJobLogs(jobId, 1, 50)
      .then((res) => setLogs(res.data))
      .catch(() => setLogs([]))
      .finally(() => setLoading(false))
  }, [open, jobId])

  const columns: ColumnsType<JobExecutionLog> = [
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (val: string) => (
        <Badge
          status={val === 'success' ? 'success' : 'error'}
          text={val}
          color={val === 'success' ? 'green' : 'red'}
        />
      ),
    },
    {
      title: 'Response Code',
      dataIndex: 'response_code',
      key: 'response_code',
    },
    {
      title: 'Duration (ms)',
      dataIndex: 'duration_ms',
      key: 'duration_ms',
    },
    {
      title: 'Error Detail',
      dataIndex: 'error_detail',
      key: 'error_detail',
      ellipsis: true,
      render: (val: string) => val || '—',
    },
    {
      title: 'Executed At',
      dataIndex: 'executed_at',
      key: 'executed_at',
      render: (val: string) => new Date(val).toLocaleString(),
    },
  ]

  return (
    <Drawer
      title="Execution Logs"
      open={open}
      onClose={onClose}
      width={720}
    >
      <Table
        rowKey="id"
        columns={columns}
        dataSource={logs}
        loading={loading}
        pagination={false}
        size="small"
      />
    </Drawer>
  )
}
