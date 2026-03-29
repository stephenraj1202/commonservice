import client from './client'

export interface FileRecord {
  id: number
  original_filename: string
  stored_filename: string
  mime_type: string
  size_bytes: number
  uploader_identity: string
  created_at: string
}

export interface PagedResponse<T> {
  total: number
  page: number
  limit: number
  data: T[]
}

export async function listFiles(page: number, limit: number): Promise<PagedResponse<FileRecord>> {
  const res = await client.get<PagedResponse<FileRecord>>('/api/v1/files', {
    params: { page, limit },
  })
  return res.data
}

export async function uploadFile(
  file: File,
  onProgress?: (pct: number) => void
): Promise<FileRecord> {
  const form = new FormData()
  form.append('file', file)
  const res = await client.post<FileRecord>('/api/v1/files/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: (e) => {
      if (onProgress && e.total) {
        onProgress(Math.round((e.loaded * 100) / e.total))
      }
    },
  })
  return res.data
}

export async function downloadFile(id: number, filename: string): Promise<void> {
  const res = await client.get(`/api/v1/files/${id}/download`, { responseType: 'blob' })
  const url = URL.createObjectURL(res.data as Blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

export async function deleteFile(id: number): Promise<void> {
  await client.delete(`/api/v1/files/${id}`)
}
