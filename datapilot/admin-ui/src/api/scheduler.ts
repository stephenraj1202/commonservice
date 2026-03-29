import client from './client'

export interface Job {
  id: number
  name: string
  cron_expression: string
  target_url: string
  http_method: string
  description: string
  status: string
  created_at: string
  updated_at: string
}

export interface JobExecutionLog {
  id: number
  job_id: number
  status: string
  response_code: number
  duration_ms: number
  error_detail: string
  executed_at: string
}

export interface PagedResponse<T> {
  total: number
  page: number
  limit: number
  data: T[]
}

export async function listJobs(
  page: number,
  limit: number,
  status?: string
): Promise<PagedResponse<Job>> {
  const res = await client.get<PagedResponse<Job>>('/api/v1/scheduler/jobs', {
    params: { page, limit, ...(status ? { status } : {}) },
  })
  return res.data
}

export async function createJob(data: Partial<Job>): Promise<Job> {
  const res = await client.post<Job>('/api/v1/scheduler/jobs', data)
  return res.data
}

export async function updateJob(id: number, data: Partial<Job>): Promise<Job> {
  const res = await client.put<Job>(`/api/v1/scheduler/jobs/${id}`, data)
  return res.data
}

export async function pauseJob(id: number): Promise<void> {
  await client.post(`/api/v1/scheduler/jobs/${id}/pause`)
}

export async function resumeJob(id: number): Promise<void> {
  await client.post(`/api/v1/scheduler/jobs/${id}/resume`)
}

export async function deleteJob(id: number): Promise<void> {
  await client.delete(`/api/v1/scheduler/jobs/${id}`)
}

export async function getJobLogs(
  id: number,
  page: number,
  limit: number
): Promise<PagedResponse<JobExecutionLog>> {
  const res = await client.get<PagedResponse<JobExecutionLog>>(
    `/api/v1/scheduler/jobs/${id}/logs`,
    { params: { page, limit } }
  )
  return res.data
}
