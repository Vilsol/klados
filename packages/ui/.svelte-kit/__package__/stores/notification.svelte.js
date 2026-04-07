class NotificationStore {
    notifications = $state([]);
    push(message, type = 'info', opts) {
        const id = crypto.randomUUID();
        this.notifications = [...this.notifications, { id, message, type, ...opts }];
        setTimeout(() => this.dismiss(id), 5000);
    }
    error(message, details) {
        this.push(message, 'error', {
            details,
            actions: [
                {
                    label: 'Copy',
                    onClick: () => navigator.clipboard.writeText(details ? `${message}\n${details}` : message),
                },
            ],
        });
    }
    dismiss(id) {
        this.notifications = this.notifications.filter((n) => n.id !== id);
    }
}
export const notificationStore = new NotificationStore();
