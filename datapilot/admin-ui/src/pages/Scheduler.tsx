import { useEffect, useState, useCallback } from 'react'
import { Typography, Button, Modal, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import {
  listJobs,
  createJob,
  pauseJob,
  resumeJob,
  deleteJob,
  type Job,
} from '../api/scheduler'
import JobForm, { type JobFormValues } from '../components/JobForm'
import JobTable from '../components/JobTable'
import LogDrawer from '../components/LogDrawer'
import { schedulerAccent } from '../theme'

const { Title } = Typography

export default function Scheduler() {
  const [jobs, setJobs] = useState<Job[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [limit, setLimit] = useState(10)
  const [loading, setLoading] = useState(false)

  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [createLoading, setCreateLoading] = useState(false)

  const [logsJobId, setLogsJobId] = useState<number | null>(null)
  const [logsOpen, setLogsOpen] = useState(false)

  const fetchJobs = useCallback(async (p: number, l: number) => {
    setLoading(true)
    try {
      const res = await listJobs(p, l)
      setJobs(res.data)
      setTotal(res.total)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchJobs(page, limit)
  }, [fetchJobs, page, limit])

  async function handleCreate(values: JobFormValues) {
    setCreateLoading(true)
    try {
      await createJob(values)
      message.success('Job created')
      setCreateModalOpen(false)
      fetchJobs(page, limit)
    } finally {
      setCreateLoading(false)
    }
  }

  async function handlePause(id: number) {
    await pauseJob(id)
    message.success('Job paused')
    setJobs((prev) =>
      prev.map((j) => (j.id === id ? { ...j, status: 'paused' } : j))
    )
  }

  async function handleResume(id: number) {
    await resumeJob(id)
    message.success('Job resumed')
    setJobs((prev) =>
      prev.map((j) => (j.id === id ? { ...j, status: 'active' } : j))
    )
  }

  async function handleDelete(id: number) {
    await deleteJob(id)
    message.success('Job deleted')
    setJobs((prev) => prev.filter((j) => j.id !== id))
    setTotal((prev) => prev - 1)
  }

  function handleViewLogs(id: number) {
    setLogsJobId(id)
    setLogsOpen(true)
  }

  function handlePageChange(newPage: number, newLimit: number) {
    setPage(newPage)
    setLimit(newLimit)
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title level={3} style={{ color: schedulerAccent, margin: 0 }}>
          Scheduler
        </Title>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setCreateModalOpen(true)}
        >
          Create Job
        </Button>
      </div>

      <JobTable
        jobs={jobs}
        total={total}
        page={page}
        limit={limit}
        loading={loading}
        onPageChange={handlePageChange}
        onPause={handlePause}
        onResume={handleResume}
        onDelete={handleDelete}
        onViewLogs={handleViewLogs}
      />

      <Modal
        title="Create Job"
        open={createModalOpen}
        onCancel={() => setCreateModalOpen(false)}
        footer={null}
        destroyOnClose
      >
        <JobForm onSubmit={handleCreate} loading={createLoading} />
      </Modal>

      <LogDrawer
        open={logsOpen}
        jobId={logsJobId}
        onClose={() => setLogsOpen(false)}
      />
    </div>
  )
}
