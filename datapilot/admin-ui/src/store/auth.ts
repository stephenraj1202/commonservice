import { create } from 'zustand'

interface AuthState {
  token: string | null
  user: string | null
  login: (token: string, user: string) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>(() => ({
  token: localStorage.getItem('datapilot_token'),
  user: localStorage.getItem('datapilot_user'),

  login: (token: string, user: string) => {
    localStorage.setItem('datapilot_token', token)
    localStorage.setItem('datapilot_user', user)
    useAuthStore.setState({ token, user })
  },

  logout: () => {
    localStorage.removeItem('datapilot_token')
    localStorage.removeItem('datapilot_user')
    useAuthStore.setState({ token: null, user: null })
    window.location.href = '/login'
  },
}))
