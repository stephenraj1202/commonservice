import { useEffect, useState, useCallback } from 'react'
import { Typography } from 'antd'
import { listFiles, deleteFile, type FileRecord } from '../api/files'
import UploadForm from '../components/UploadForm'
import FileTable from '../components/FileTable'
import { filesAccent } from '../theme'

const { Title } = Typography

export default function Files() {
  const [files, setFiles] = useState<FileRecord[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [limit, setLimit] = useState(10)
  const [loading, setLoading] = useState(false)

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
  }

  function handlePageChange(newPage: number, newLimit: number) {
    setPage(newPage)
    setLimit(newLimit)
  }

  return (
    <div>
      <Title level={3} style={{ color: filesAccent, marginBottom: 24 }}>
        Files
      </Title>
      <UploadForm onSuccess={handleUploadSuccess} />
      <FileTable
        files={files}
        total={total}
        page={page}
        limit={limit}
        loading={loading}
        onPageChange={handlePageChange}
        onDelete={handleDelete}
      />
    </div>
  )
}
