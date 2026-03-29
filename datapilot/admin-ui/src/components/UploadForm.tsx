import { useState } from 'react'
import { Upload, Progress, message, Space } from 'antd'
import { InboxOutlined } from '@ant-design/icons'
import type { UploadRequestOption } from 'rc-upload/lib/interface'
import { uploadFile, type FileRecord } from '../api/files'

interface UploadFormProps {
  onSuccess: (file: FileRecord) => void
}

interface FileProgress {
  name: string
  progress: number
}

export default function UploadForm({ onSuccess }: UploadFormProps) {
  const [uploads, setUploads] = useState<Map<string, FileProgress>>(new Map())

  async function handleUpload({ file }: UploadRequestOption) {
    const fileName = (file as File).name
    
    setUploads(prev => new Map(prev).set(fileName, { name: fileName, progress: 0 }))
    
    try {
      const record = await uploadFile(file as File, (pct) => {
        setUploads(prev => new Map(prev).set(fileName, { name: fileName, progress: pct }))
      })
      message.success(`${fileName} uploaded successfully`)
      onSuccess(record)
    } catch {
      message.error(`Failed to upload ${fileName}`)
    } finally {
      setTimeout(() => {
        setUploads(prev => {
          const next = new Map(prev)
          next.delete(fileName)
          return next
        })
      }, 2000)
    }
  }

  return (
    <div style={{ marginBottom: 24 }}>
      <Upload.Dragger
        customRequest={handleUpload}
        showUploadList={false}
        multiple={true}
      >
        <p className="ant-upload-drag-icon">
          <InboxOutlined />
        </p>
        <p className="ant-upload-text">Click or drag files here to upload</p>
        <p className="ant-upload-hint">Supports multiple files, any type up to 100 MB each</p>
      </Upload.Dragger>
      {uploads.size > 0 && (
        <Space direction="vertical" style={{ width: '100%', marginTop: 12 }}>
          {Array.from(uploads.values()).map((upload) => (
            <div key={upload.name}>
              <div style={{ marginBottom: 4, fontSize: 12, color: '#666' }}>
                {upload.name}
              </div>
              <Progress percent={upload.progress} />
            </div>
          ))}
        </Space>
      )}
    </div>
  )
}
