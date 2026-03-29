import { useEffect, useState, useCallback } from 'react'
import { Typography, Button, Space, message, Segmented, Card, Row, Col, Checkbox, Popconfirm } from 'antd'
import { AppstoreOutlined, UnorderedListOutlined, DeleteOutlined, FileTextOutlined, FilePdfOutlined, FileImageOutlined, FileZipOutlined, FileExcelOutlined, FileWordOutlined, FileOutlined } from '@ant-design/icons'
import { listFiles, deleteFile, type FileRecord } from '../api/files'
import UploadForm from '../components/UploadForm'
import FileTable from '../components/FileTable'
import { filesAccent } from '../theme'

const { Title } = Typography

function getFileIcon(filename: string) {
  const ext = filename.split('.').pop()?.toLowerCase()
  const iconProps = { style: { fontSize: 48 } }
  
  if (['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp'].includes(ext || '')) {
    return <FileImageOutlined {...iconProps} style={{ ...iconProps.style, color: '#52c41a' }} />
  }
  if (ext === 'pdf') {
    return <FilePdfOutlined {...iconProps} style={{ ...iconProps.style, color: '#ff4d4f' }} />
  }
  if (['doc', 'docx'].includes(ext || '')) {
    return <FileWordOutlined {...iconProps} style={{ ...iconProps.style, color: '#1890ff' }} />
  }
  if (['xls', 'xlsx', 'csv'].includes(ext || '')) {
    return <FileExcelOutlined {...iconProps} style={{ ...iconProps.style, color: '#52c41a' }} />
  }
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(ext || '')) {
    return <FileZipOutlined {...iconProps} style={{ ...iconProps.style, color: '#faad14' }} />
  }
  if (['txt', 'log', 'md'].includes(ext || '')) {
    return <FileTextOutlined {...iconProps} style={{ ...iconProps.style, color: '#8c8c8c' }} />
  }
  return <FileOutlined {...iconProps} style={{ ...iconProps.style, color: '#722ed1' }} />
}

export default function Files() {
  const [files, setFiles] = useState<FileRecord[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [limit, setLimit] = useState(10)
  const [loading, setLoading] = useState(false)
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('list')
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())

  const fetchFiles = useCallback(async (p: number, l: number) => {
    setLoading(true)
    try {
      const res = await listFiles(p, l)
      setFiles(res.data)
      setTotal(res.total)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchFiles(page, limit)
  }, [fetchFiles, page, limit])

  function handleUploadSuccess(file: FileRecord) {
    setFiles((prev) => [file, ...prev])
    setTotal((prev) => prev + 1)
  }

  async function handleDelete(id: number) {
    await deleteFile(id)
    setFiles((prev) => prev.filter((f) => f.id !== id))
    setTotal((prev) => prev - 1)
    setSelectedIds(prev => {
      const next = new Set(prev)
      next.delete(id)
      return next
    })
  }

  async function handleBulkDelete() {
    try {
      await Promise.all(Array.from(selectedIds).map(id => deleteFile(id)))
      setFiles(prev => prev.filter(f => !selectedIds.has(f.id)))
      setTotal(prev => prev - selectedIds.size)
      setSelectedIds(new Set())
      message.success(`Deleted ${selectedIds.size} files`)
    } catch {
      message.error('Failed to delete some files')
    }
  }

  function handlePageChange(newPage: number, newLimit: number) {
    setPage(newPage)
    setLimit(newLimit)
    setSelectedIds(new Set())
  }

  function toggleSelect(id: number) {
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  function toggleSelectAll() {
    if (selectedIds.size === files.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(files.map(f => f.id)))
    }
  }

  return (
    <div>
      <Title level={3} style={{ color: filesAccent, marginBottom: 24 }}>
        File Management
      </Title>

      <UploadForm onSuccess={handleUploadSuccess} />

      <Space style={{ marginBottom: 16 }}>
        <Segmented
          value={viewMode}
          onChange={(value) => setViewMode(value as 'grid' | 'list')}
          options={[
            { label: 'List', value: 'list', icon: <UnorderedListOutlined /> },
            { label: 'Grid', value: 'grid', icon: <AppstoreOutlined /> },
          ]}
        />
        {selectedIds.size > 0 && (
          <Popconfirm
            title={`Delete ${selectedIds.size} selected files?`}
            onConfirm={handleBulkDelete}
            okText="Yes"
            cancelText="No"
          >
            <Button danger icon={<DeleteOutlined />}>
              Delete Selected ({selectedIds.size})
            </Button>
          </Popconfirm>
        )}
        {viewMode === 'list' && files.length > 0 && (
          <Checkbox
            checked={selectedIds.size === files.length}
            indeterminate={selectedIds.size > 0 && selectedIds.size < files.length}
            onChange={toggleSelectAll}
          >
            Select All
          </Checkbox>
        )}
      </Space>

      {viewMode === 'grid' ? (
        <Row gutter={[16, 16]}>
          {files.map((file) => (
            <Col xs={24} sm={12} md={8} lg={6} key={file.id}>
              <Card
                hoverable
                style={{ position: 'relative' }}
                cover={
                  <div style={{ padding: 24, textAlign: 'center', background: '#fafafa' }}>
                    {getFileIcon(file.original_filename)}
                  </div>
                }
                actions={[
                  <Checkbox
                    checked={selectedIds.has(file.id)}
                    onChange={() => toggleSelect(file.id)}
                  />,
                  <Popconfirm
                    title="Delete this file?"
                    onConfirm={() => handleDelete(file.id)}
                    okText="Yes"
                    cancelText="No"
                  >
                    <DeleteOutlined key="delete" style={{ color: '#ff4d4f' }} />
                  </Popconfirm>,
                ]}
              >
                <Card.Meta
                  title={
                    <div style={{ 
                      overflow: 'hidden', 
                      textOverflow: 'ellipsis', 
                      whiteSpace: 'nowrap' 
                    }}>
                      {file.original_filename}
                    </div>
                  }
                  description={
                    <div>
                      <div>{(file.size_bytes / 1024 / 1024).toFixed(2)} MB</div>
                      <div style={{ fontSize: 11, color: '#999' }}>
                        {new Date(file.created_at).toLocaleDateString()}
                      </div>
                    </div>
                  }
                />
              </Card>
            </Col>
          ))}
        </Row>
      ) : (
        <FileTable
          files={files}
          total={total}
          page={page}
          limit={limit}
          loading={loading}
          onDelete={handleDelete}
          onPageChange={handlePageChange}
          selectedIds={selectedIds}
          onToggleSelect={toggleSelect}
        />
      )}
    </div>
  )
}
