// PluginContext is passed to every UI component as the `ctx` prop.
export interface PluginContext {
  cluster: {
    name: string
    version: string
  }
  namespace: string
  k8s?: K8sContext
  logs?: LogsContext
  exec?: ExecContext
  storage?: StorageContext
  events?: EventsContext
}

// K8sContext is present when the plugin declares resource permissions.
export interface K8sContext {
  list(gvr: string, namespace?: string): Promise<KubernetesObject[]>
  get(gvr: string, namespace: string, name: string): Promise<KubernetesObject>
  watch(gvr: string, namespace: string, cb: (event: WatchEvent) => void): () => void
}

export interface LogsContext {
  stream(namespace: string, podName: string, container: string, cb: (line: string) => void): () => void
}

export interface ExecContext {
  open(namespace: string, podName: string, container: string): Promise<ExecSession>
}

export interface StorageContext {
  get(key: string): Promise<string | null>
  set(key: string, value: string): Promise<void>
  delete(key: string): Promise<void>
  list(): Promise<string[]>
}

export interface EventsContext {
  subscribe(eventName: string, cb: (payload: unknown) => void): () => void
}

// KubernetesObject is an unstructured Kubernetes object.
export interface KubernetesObject {
  apiVersion?: string
  kind?: string
  metadata?: ObjectMeta
  spec?: Record<string, unknown>
  status?: Record<string, unknown>
  [key: string]: unknown
}

export interface ObjectMeta {
  name?: string
  namespace?: string
  labels?: Record<string, string>
  annotations?: Record<string, string>
  uid?: string
  resourceVersion?: string
  creationTimestamp?: string
  [key: string]: unknown
}

export interface WatchEvent {
  type: 'ADDED' | 'MODIFIED' | 'DELETED'
  object: KubernetesObject
}

export interface ExecSession {
  send(data: string): void
  resize(cols: number, rows: number): void
  close(): void
  onData(cb: (data: string) => void): void
}

// DetailTabProps are the props passed to a detail tab Svelte component.
export interface DetailTabProps {
  resource: KubernetesObject
  ctx: PluginContext
}

// CommandProps are the props passed to a command Svelte component.
export interface CommandProps {
  ctx: PluginContext
}
