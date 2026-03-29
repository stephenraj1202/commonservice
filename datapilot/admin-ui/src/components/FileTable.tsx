import { useState } from 'react'
import { Table, Button, Space } from 'antd'
import { DownloadOutlined, DeleteOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { downloadFile, type FileRecord } from '../api/files'
import ConfirmDialog from './ConfirmDialog'

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

interface FileTableProps {
  files: FileRecord[]
  total: number
  page: number
  limit: number
  loading: boolean
  onPageChange: (page: number, limit: number) => void
  onDelete: (id: number) => Promise<void>
}

export default function FileTable({
  files,
  total,
  page,
  limit,
  loading,
  onPageChange,
  onDelete,
}: FileTableProps) {
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

  const columns: ColumnsType<FileRecord> = [
    {
      title: 'Filename',
      dataIndex: 'original_filename',
      key: 'original_filename',
      ellipsis: true,
    },
    {
      title: 'MIME Type',
      dataIndex: 'mime_type',
      key: 'mime_type',
    },
    {
      title: 'Size',
      dataIndex: 'size_bytes',
      key: 'size_bytes',
      render: (val: number) => formatBytes(val),
    },
    {
      title: 'Uploader',
      dataIndex: 'uploader_identity',
      key: 'uploader_identity',
    },
    {
      title: 'Upload Date',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (val: string) => new Date(val).toLocaleString(),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Button
            icon={<DownloadOutlined />}
            size="small"
            onClick={() => downloadFile(record.id, record.original_filename)}
          >
            Download
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
        dataSource={files}
        loading={loading}
        pagination={{
          current: page,
          pageSize: limit,
          total,
          showSizeChanger: true,
          onChange: onPageChange,
        }}
      />
      <ConfirmDialog
        open={confirmId !== null}
        title="Delete File"
        message="Are you sure you want to delete this file? This action cannot be undone."
        onConfirm={handleConfirmDelete}
        onCancel={() => setConfirmId(null)}
        loading={deleting}
      />
    </>
  )
}
