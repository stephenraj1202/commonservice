import { Card, Statistic } from 'antd'

interface StatCardProps {
  title: string
  value: number | string
  icon: React.ReactNode
  color: string
}

export default function StatCard({ title, value, icon, color }: StatCardProps) {
  return (
    <Card>
      <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
        <div style={{ fontSize: 32, color }}>{icon}</div>
        <Statistic title={title} value={value} valueStyle={{ color }} />
      </div>
    </Card>
  )
}
