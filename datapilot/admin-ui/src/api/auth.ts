import client from './client'

export async function login(username: string, password: string): Promise<{ token: string }> {
  const res = await client.post<{ token: string }>('/api/v1/auth/login', { username, password })
  return res.data
}

export async function register(username: string, password: string): Promise<void> {
  await client.post('/api/v1/auth/register', { username, password })
}
