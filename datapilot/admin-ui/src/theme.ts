import type { ThemeConfig } from 'antd'

export const tealColor = '#0d9488'
export const amberColor = '#d97706'

export const theme: ThemeConfig = {
  token: {
    colorPrimary: '#6366f1',
  },
  components: {
    Card: {
      // Component-level overrides are applied per-section via inline styles
    },
  },
}

/** Accent color for the Files section */
export const filesAccent = tealColor

/** Accent color for the Scheduler section */
export const schedulerAccent = amberColor
