import { useState } from 'react'
import { Table, Button, Space, Badge } from 'antd'
import { PauseCircleOutlined, PlayCircleOutlined, DeleteOutlined, FileTextOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { Job } from '../api/scheduler'
import ConfirmDialog from './ConfirmDialog'

interface JobTableProps {
  jobs: Job[]
  total: number
  page: number
  limit: number
  loading: boolean
  onPageChange: (page: number, limit: number) => void
  onPause: (id: number) => Promise<void>
  onResume: (id: number) => Promise<void>
  onDelete: (id: number) => Promise<void>
  onViewLogs: (id: number) => void
}

function statusBadge(status: string) {
  if (status === 'active') return <Badge status="success" text="active" />
  if (status === 'paused') return <Badge status="warning" text="paused" />
  return <Badge status="error" text={status} />
}

export default function JobTable({
  jobs,
  total,
  page,
  limit,
  loading,
  onPageChange,
  onPause,
  onResume,
  onDelete,
  onViewLogs,
}: JobTableProps) {
  const [confirmId, setConfirmId] = useState<number | null>(null)
  const [deleting, setDeleting] = useState(false)

  async function handleConfirmDelete() {
    if (confirmId === null) return
    setDeleting(true)
    try {
      await onDelete(confirmId)
    } finally {
      setDeleting(false)
      setConfirmId(null)
    }
  }

  const columns: ColumnsType<Job> = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
    },
    {
      title: 'Cron Expression',
      dataIndex: 'cron_expression',
      key: 'cron_expression',
    },
    {
      title: 'Target URL',
      dataIndex: 'target_url',
      key: 'target_url',
      ellipsis: true,
    },
    {
      title: 'Method',
      dataIndex: 'http_method',
      key: 'http_method',
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (val: string) => statusBadge(val),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record) => (
        <Space>
          {record.status === 'active' && (
            <Button
              icon={<PauseCircleOutlined />}
              size="small"
              onClick={() => onPause(record.id)}
            >
              Pause
            </Button>
          )}
          {record.status === 'paused' && (
            <Button
              icon={<PlayCircleOutlined />}
              size="small"
              onClick={() => onResume(record.id)}
            >
              Resume
            </Button>
          )}
          <Button
            icon={<FileTextOutlined />}
            size="small"
            onClick={() => onViewLogs(record.id)}
          >
            Logs
          </Button>
          <Button
            icon={<DeleteOutlined />}
            size="small"
            danger
            onClick={() => setConfirmId(record.id)}
          >
            Delete
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <>
      <Table
        rowKey="id"
        columns={columns}
        dataSource={jobs}
        loading={loading}
        rowClassName={(record) =>
          record.status === 'failed' ? 'job-row-failed' : ''
        }
        pagination={{
          current: page,
          pageSize: limit,
          total,
          showSizeChanger: true,
          onChange: onPageChange,
        }}
      />
      <style>{`
        .job-row-failed td {
          background-color: #fef3c7 !important;
        }
        .job-row-failed:hover td {
          background-color: #fde68a !important;
        }
      `}</style>
      <ConfirmDialog
        open={confirmId !== null}
        title="Delete Job"
        message="Are you sure you want to delete this job? This action cannot be undone."
        onConfirm={handleConfirmDelete}
        onCancel={() => setConfirmId(null)}
        loading={deleting}
      />
    </>
  )
}
