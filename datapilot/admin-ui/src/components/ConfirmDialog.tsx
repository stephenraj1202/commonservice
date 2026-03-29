import { Modal } from 'antd'

interface ConfirmDialogProps {
  open: boolean
  title: string
  message: string
  onConfirm: () => void
  onCancel: () => void
  loading?: boolean
}

export default function ConfirmDialog({
  open,
  title,
  message,
  onConfirm,
  onCancel,
  loading = false,
}: ConfirmDialogProps) {
  return (
    <Modal
      open={open}
      title={title}
      onOk={onConfirm}
      onCancel={onCancel}
      okButtonProps={{ loading, danger: true }}
      okText="Confirm"
      cancelText="Cancel"
    >
      <p>{message}</p>
    </Modal>
  )
}
