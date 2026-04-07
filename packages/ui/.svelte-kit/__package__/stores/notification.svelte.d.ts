export interface ToastAction {
    label: string;
    onClick: () => void;
}
export interface Toast {
    id: string;
    message: string;
    type: 'info' | 'success' | 'error';
    details?: string;
    actions?: ToastAction[];
}
declare class NotificationStore {
    notifications: Toast[];
    push(message: string, type?: Toast['type'], opts?: Pick<Toast, 'details' | 'actions'>): void;
    error(message: string, details?: string): void;
    dismiss(id: string): void;
}
export declare const notificationStore: NotificationStore;
export {};
