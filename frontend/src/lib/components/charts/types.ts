export interface TimeSeriesPoint {
  t: number
  v: number
}

export interface TimeSeries {
  labels: Record<string, string>
  points: TimeSeriesPoint[]
}

export interface MetricResult {
  name: string
  unit: string
  series: TimeSeries[]
}

export interface ThresholdLine {
  label: string
  series: TimeSeriesPoint[]
}

export interface Annotation {
  t: number
  label: string
  severity: string
}

export interface MetricsResponse {
  metrics: MetricResult[]
  thresholds: ThresholdLine[]
  annotations: Annotation[]
}

export interface MetricsCapability {
  hasMetricsServer: boolean
  hasPrometheus: boolean
  prometheusUrl?: string
}
