import { useState, useEffect } from 'react'
import { Typography, Card, Space, Button, Input, message, Divider, Tabs } from 'antd'
import { CopyOutlined, KeyOutlined } from '@ant-design/icons'
import { listFiles, type FileRecord } from '../api/files'
import { listJobs, type Job } from '../api/scheduler'

const { Title, Text, Paragraph } = Typography
const { TextArea } = Input

export default function Documentation() {
  const [files, setFiles] = useState<FileRecord[]>([])
  const [jobs, setJobs] = useState<Job[]>([])
  const [token, setToken] = useState('')

  useEffect(() => {
    const storedToken = localStorage.getItem('datapilot_token') || ''
    setToken(storedToken)
    
    async function fetchData() {
      try {
        const [filesRes, jobsRes] = await Promise.all([
          listFiles(1, 100),
          listJobs(1, 100)
        ])
        setFiles(filesRes.data || [])
        setJobs(jobsRes.data || [])
      } catch (err) {
        console.error('Failed to fetch data:', err)
      }
    }
    fetchData()
  }, [])

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    message.success('Copied to clipboard!')
  }

  const loginCommand = `# Login and get token
curl -X POST https://commonservice.datapilot.co.in/api/v1/auth/login \\
  -H "Content-Type: application/json" \\
  -d '{"username":"your_username","password":"your_password"}'

# Save token to variable
TOKEN="your_token_here"`

  const items = [
    {
      key: 'auth',
      label: 'Authentication',
      children: (
        <Space direction="vertical" style={{ width: '100%' }}>
          <Card title="Generate Token" size="small">
            <Paragraph>
              <Text strong>Your Current Token:</Text>
              <TextArea
                value={token}
                readOnly
                autoSize={{ minRows: 2, maxRows: 4 }}
                style={{ marginTop: 8, fontFamily: 'monospace', fontSize: 12 }}
              />
              <Button
                icon={<CopyOutlined />}
                onClick={() => copyToClipboard(token)}
                style={{ marginTop: 8 }}
              >
                Copy Token
              </Button>
            </Paragraph>
          </Card>

          <Card title="Login Command" size="small">
            <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, overflow: 'auto' }}>
              {loginCommand}
            </pre>
            <Button icon={<CopyOutlined />} onClick={() => copyToClipboard(loginCommand)}>
              Copy Command
            </Button>
          </Card>
        </Space>
      ),
    },
    {
      key: 'files',
      label: `Files (${files.length})`,
      children: (
        <Space direction="vertical" style={{ width: '100%' }}>
          <Card title="Download All Files" size="small">
            <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, overflow: 'auto' }}>
{`TOKEN="${token}"

# Download all files
${files.map(f => `curl -X GET "https://commonservice.datapilot.co.in/api/v1/files/${f.id}/download" \\
  -H "Authorization: Bearer $TOKEN" \\
  -o "${f.original_filename}"`).join('\n\n')}`}
            </pre>
            <Button
              icon={<CopyOutlined />}
              onClick={() => copyToClipboard(`TOKEN="${token}"\n\n${files.map(f => 
                `curl -X GET "https://commonservice.datapilot.co.in/api/v1/files/${f.id}/download" \\\n  -H "Authorization: Bearer $TOKEN" \\\n  -o "${f.original_filename}"`
              ).join('\n\n')}`)}
            >
              Copy All Download Commands
            </Button>
          </Card>

          <Divider>Individual Files</Divider>

          {files.map(file => (
            <Card
              key={file.id}
              title={file.original_filename}
              size="small"
              extra={
                <Text type="secondary">
                  {(file.size_bytes / 1024 / 1024).toFixed(2)} MB
                </Text>
              }
            >
              <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, overflow: 'auto', fontSize: 12 }}>
{`curl -X GET "https://commonservice.datapilot.co.in/api/v1/files/${file.id}/download" \\
  -H "Authorization: Bearer ${token}" \\
  -o "${file.original_filename}"`}
              </pre>
              <Button
                icon={<CopyOutlined />}
                size="small"
                onClick={() => copyToClipboard(
                  `curl -X GET "https://commonservice.datapilot.co.in/api/v1/files/${file.id}/download" \\\n  -H "Authorization: Bearer ${token}" \\\n  -o "${file.original_filename}"`
                )}
              >
                Copy Command
              </Button>
            </Card>
          ))}
        </Space>
      ),
    },
    {
      key: 'jobs',
      label: `Jobs (${jobs.length})`,
      children: (
        <Space direction="vertical" style={{ width: '100%' }}>
          <Card title="List All Jobs" size="small">
            <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, overflow: 'auto' }}>
{`curl -X GET "https://commonservice.datapilot.co.in/api/v1/scheduler/jobs" \\
  -H "Authorization: Bearer ${token}"`}
            </pre>
            <Button
              icon={<CopyOutlined />}
              onClick={() => copyToClipboard(
                `curl -X GET "https://commonservice.datapilot.co.in/api/v1/scheduler/jobs" \\\n  -H "Authorization: Bearer ${token}"`
              )}
            >
              Copy Command
            </Button>
          </Card>

          {jobs.length > 0 && (
            <>
              <Divider>Individual Jobs</Divider>
              {jobs.map(job => (
                <Card
                  key={job.id}
                  title={job.name}
                  size="small"
                  extra={<Text type="secondary">{job.cron_expression}</Text>}
                >
                  <Space direction="vertical" style={{ width: '100%' }}>
                    <div>
                      <Text strong>Get Job Details:</Text>
                      <pre style={{ background: '#f5f5f5', padding: 8, borderRadius: 4, fontSize: 11, marginTop: 4 }}>
{`curl -X GET "https://commonservice.datapilot.co.in/api/v1/scheduler/jobs/${job.id}" \\
  -H "Authorization: Bearer ${token}"`}
                      </pre>
                    </div>
                    <div>
                      <Text strong>Pause Job:</Text>
                      <pre style={{ background: '#f5f5f5', padding: 8, borderRadius: 4, fontSize: 11, marginTop: 4 }}>
{`curl -X POST "https://commonservice.datapilot.co.in/api/v1/scheduler/jobs/${job.id}/pause" \\
  -H "Authorization: Bearer ${token}"`}
                      </pre>
                    </div>
                    <div>
                      <Text strong>Resume Job:</Text>
                      <pre style={{ background: '#f5f5f5', padding: 8, borderRadius: 4, fontSize: 11, marginTop: 4 }}>
{`curl -X POST "https://commonservice.datapilot.co.in/api/v1/scheduler/jobs/${job.id}/resume" \\
  -H "Authorization: Bearer ${token}"`}
                      </pre>
                    </div>
                  </Space>
                </Card>
              ))}
            </>
          )}
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Title level={3} style={{ marginBottom: 24 }}>
        <KeyOutlined /> API Documentation & Commands
      </Title>
      <Tabs items={items} />
    </div>
  )
}
