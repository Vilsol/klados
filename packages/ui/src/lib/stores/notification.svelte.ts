export interface ToastAction {
  label: string
  onClick: () => void
}

export interface Toast {
  id: string
  message: string
  type: 'info' | 'success' | 'error'
  details?: string
  actions?: ToastAction[]
}

class NotificationStore {
  notifications = $state<Toast[]>([])

  push(message: string, type: Toast['type'] = 'info', opts?: Pick<Toast, 'details' | 'actions'>) {
    const id = crypto.randomUUID()
    this.notifications = [...this.notifications, { id, message, type, ...opts }]
    setTimeout(() => this.dismiss(id), 5000)
  }

  success(message: string, details?: string) {
    this.push(message, 'success', { details })
  }

  error(message: string, details?: string) {
    this.push(message, 'error', {
      details,
      actions: [
        {
          label: 'Copy',
          onClick: () => navigator.clipboard.writeText(details ? `${message}\n${details}` : message),
        },
      ],
    })
  }

  dismiss(id: string) {
    this.notifications = this.notifications.filter((n) => n.id !== id)
  }
}

export const notificationStore = new NotificationStore()
