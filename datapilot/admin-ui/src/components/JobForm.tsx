import { useEffect, useState } from 'react'
import { Form, Input, Select, Button } from 'antd'
import cronstrue from 'cronstrue'
import type { Job } from '../api/scheduler'

export interface JobFormValues {
  name: string
  cron_expression: string
  target_url: string
  http_method: string
  description?: string
}

interface JobFormProps {
  initialValues?: Partial<Job>
  onSubmit: (values: JobFormValues) => Promise<void>
  loading?: boolean
}

export default function JobForm({ initialValues, onSubmit, loading = false }: JobFormProps) {
  const [form] = Form.useForm<JobFormValues>()
  const [cronDescription, setCronDescription] = useState<string>('')
  const [cronError, setCronError] = useState<string>('')

  useEffect(() => {
    if (initialValues) {
      form.setFieldsValue(initialValues)
      if (initialValues.cron_expression) {
        updateCronDescription(initialValues.cron_expression)
      }
    }
  }, [initialValues, form])

  function updateCronDescription(expr: string) {
    if (!expr) {
      setCronDescription('')
      setCronError('')
      return
    }
    try {
      const desc = cronstrue.toString(expr)
      setCronDescription(desc)
      setCronError('')
    } catch {
      setCronDescription('')
      setCronError('Invalid cron expression')
    }
  }

  async function handleFinish(values: JobFormValues) {
    await onSubmit(values)
    form.resetFields()
    setCronDescription('')
    setCronError('')
  }

  return (
    <Form
      form={form}
      layout="vertical"
      onFinish={handleFinish}
      initialValues={{ http_method: 'GET' }}
    >
      <Form.Item
        label="Name"
        name="name"
        rules={[{ required: true, message: 'Job name is required' }]}
      >
        <Input placeholder="My scheduled job" />
      </Form.Item>

      <Form.Item
        label="Cron Expression"
        name="cron_expression"
        rules={[{ required: true, message: 'Cron expression is required' }]}
      >
        <Input
          placeholder="0 * * * *"
          onChange={(e) => updateCronDescription(e.target.value)}
        />
      </Form.Item>

      {cronDescription && (
        <div style={{ marginTop: -16, marginBottom: 16, color: '#6366f1', fontSize: 13 }}>
          {cronDescription}
        </div>
      )}
      {cronError && (
        <div style={{ marginTop: -16, marginBottom: 16, color: '#ef4444', fontSize: 13 }}>
          {cronError}
        </div>
      )}

      <Form.Item
        label="Target URL"
        name="target_url"
        rules={[
          { required: true, message: 'Target URL is required' },
          { type: 'url', message: 'Must be a valid URL' },
        ]}
      >
        <Input placeholder="https://example.com/webhook" />
      </Form.Item>

      <Form.Item
        label="HTTP Method"
        name="http_method"
        rules={[{ required: true }]}
      >
        <Select>
          <Select.Option value="GET">GET</Select.Option>
          <Select.Option value="POST">POST</Select.Option>
          <Select.Option value="PUT">PUT</Select.Option>
          <Select.Option value="DELETE">DELETE</Select.Option>
          <Select.Option value="PATCH">PATCH</Select.Option>
        </Select>
      </Form.Item>

      <Form.Item label="Description" name="description">
        <Input.TextArea rows={3} placeholder="Optional description" />
      </Form.Item>

      <Form.Item>
        <Button type="primary" htmlType="submit" loading={loading} block>
          {initialValues?.id ? 'Update Job' : 'Create Job'}
        </Button>
      </Form.Item>
    </Form>
  )
}
