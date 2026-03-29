import { useState } from 'react'
import { Upload, Progress, message } from 'antd'
import { InboxOutlined } from '@ant-design/icons'
import type { UploadRequestOption } from 'rc-upload/lib/interface'
import { uploadFile, type FileRecord } from '../api/files'

interface UploadFormProps {
  onSuccess: (file: FileRecord) => void
}

export default function UploadForm({ onSuccess }: UploadFormProps) {
  const [progress, setProgress] = useState<number | null>(null)

  async function handleUpload({ file }: UploadRequestOption) {
    setProgress(0)
    try {
      const record = await uploadFile(file as File, (pct) => setProgress(pct))
      message.success(`${(file as File).name} uploaded successfully`)
      onSuccess(record)
    } catch {
      message.error(`Failed to upload ${(file as File).name}`)
    } finally {
      setProgress(null)
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
      {progress !== null && (
        <Progress percent={progress} style={{ marginTop: 12 }} />
      )}
    </div>
  )
}
