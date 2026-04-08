<script lang="ts">
  import { untrack } from 'svelte'
  import uPlot from 'uplot'
  import 'uplot/dist/uPlot.min.css'
  import type { TimeSeries, ThresholdLine, Annotation } from './types'
  import { getFormatter } from './units'

  interface ZoomRange { min: number; max: number }

  interface Props {
    title: string
    unit: string
    series: TimeSeries[]
    thresholds?: ThresholdLine[]
    annotations?: Annotation[]
    loading?: boolean
    height?: number
    forceZero?: boolean
    zoomRange?: ZoomRange | null
    onzoom?: (range: ZoomRange | null) => void
  }

  let { title, unit, series, thresholds = [], annotations = [], loading = false, height = 200, forceZero = true, zoomRange = null, onzoom }: Props = $props()

  const COLORS = [
    '#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#8b5cf6',
    '#06b6d4', '#ec4899', '#84cc16', '#f97316', '#14b8a6',
  ]

  let container: HTMLDivElement = $state(null as unknown as HTMLDivElement)
  let chart: uPlot | null = null
  let tooltip: HTMLDivElement = $state(null as unknown as HTMLDivElement)
  let legendToggles: boolean[] = $state([])
  // Flag to suppress onzoom callback when we're applying an externally-driven scale change
  let isProgrammaticScale = false

  // Pre-sorted annotations for binary-search during drawAxes
  const sortedAnnotations = $derived([...annotations].sort((a, b) => a.t - b.t))

  const ANNOTATION_COLORS: Record<string, string> = {
    error: '#ef4444',
    warning: '#f59e0b',
    info: '#3b82f6',
  }

  // Sync legendToggles length with series, preserving user choices
  $effect(() => {
    const len = series.length
    untrack(() => {
      legendToggles = Array.from({ length: len }, (_, i) => legendToggles[i] !== false)
    })
  })

  function toColumnar(series: TimeSeries[]): uPlot.AlignedData {
    if (!series.length || !series[0].points.length) {
      return [new Float64Array(), ...series.map(() => new Float64Array())]
    }

    // Use first series timestamps as x-axis
    const timestamps = new Float64Array(series[0].points.map((p) => p.t))
    const values = series.map((s) => new Float64Array(s.points.map((p) => p.v)))
    return [timestamps, ...values]
  }

  function seriesLabel(s: TimeSeries, i: number): string {
    return (
      s.labels['container'] ||
      s.labels['pod'] ||
      s.labels['node'] ||
      Object.values(s.labels).find((v) => v) ||
      (series.length === 1 ? title : `series${i + 1}`)
    )
  }

  function buildSeriesDefs(series: TimeSeries[], toggles: boolean[]): uPlot.Series[] {
    return [
      {},
      ...series.map((s, i) => ({
        label: seriesLabel(s, i),
        stroke: COLORS[i % COLORS.length],
        width: 1.5,
        show: toggles[i] !== false,
        points: { show: false },
      })),
    ]
  }

  $effect(() => {
    if (!container || loading) return

    // Read series with untrack so data changes don't recreate the chart — setData handles updates
    const initialSeries = untrack(() => series)
    const fmt = getFormatter(unit)

    const style = getComputedStyle(container)
    const mutedColor = style.getPropertyValue('--color-muted').trim() || '#64748b'
    const fgColor = style.getPropertyValue('--color-fg').trim() || '#0f172a'
    const borderColor = style.getPropertyValue('--color-border').trim() || '#e2e8f0'

    const opts: uPlot.Options = {
      width: container.clientWidth || 400,
      height,
      series: buildSeriesDefs(initialSeries, untrack(() => legendToggles)),
      axes: [
        {
          stroke: mutedColor,
          ticks: { stroke: borderColor },
          grid: { stroke: borderColor },
          space: 80,
          values: (_u, vals) => vals.map((v) => new Date(v * 1000).toLocaleTimeString()),
        },
        {
          stroke: mutedColor,
          ticks: { stroke: borderColor },
          grid: { stroke: borderColor },
          size: 90,
          values: (_u, vals) => vals.map((v) => (v != null ? fmt(v) : '')),
        },
      ],
      scales: forceZero ? {
        y: { range: (_u, _min, max) => {
          const thresholdMax = thresholds.reduce((acc, t) => Math.max(acc, t.series.at(-1)?.v ?? 0), 0)
          return [0, Math.max(max || 1, thresholdMax) * 1.05]
        }},
      } : {},
      legend: { show: false },
      cursor: {},
      hooks: {
        setCursor: [
          (u) => {
            if (!tooltip) return
            const idx = u.cursor.idx
            if (idx == null) {
              tooltip.style.display = 'none'
              return
            }

            const x = u.cursor.left ?? 0
            const y = u.cursor.top ?? 0

            // Find which series the cursor is vertically closest to
            const cursorY = u.cursor.top ?? 0
            let closestSeries = -1
            let minDist = Infinity
            for (let i = 1; i < u.series.length; i++) {
              const s = u.series[i]
              if (!s.show) continue
              const val = u.data[i][idx] as number | null
              if (val == null) continue
              const seriesY = u.valToPos(val, s.scale ?? 'y')
              const dist = Math.abs(seriesY - cursorY)
              if (dist < minDist) { minDist = dist; closestSeries = i }
            }

            // Build visible series list and sort by descending value (matches chart visual order)
            const visibleSeries = []
            for (let i = 1; i < u.series.length; i++) {
              const s = u.series[i]
              if (!s.show) continue
              visibleSeries.push({ i, s, val: u.data[i][idx] as number | null, color: COLORS[(i - 1) % COLORS.length] })
            }
            visibleSeries.sort((a, b) => (b.val ?? -Infinity) - (a.val ?? -Infinity))

            // Check if cursor is near an annotation marker
            const cursorTime = u.data[0][idx] as number
            const ann = sortedAnnotations
            const nearAnnotations = ann.filter((a) => Math.abs(u.valToPos(a.t, 'x') - (u.cursor.left ?? 0)) < 8)

            let html = `<div style="font-size:11px;color:${fgColor}">`
            if (nearAnnotations.length) {
              for (const a of nearAnnotations) {
                const color = ANNOTATION_COLORS[a.severity] ?? '#3b82f6'
                html += `<div style="display:flex;align-items:center;gap:6px;margin-bottom:4px;padding-bottom:4px;border-bottom:1px solid ${borderColor}">`
                html += `<span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${color};flex-shrink:0"></span>`
                html += `<span style="color:${color};font-weight:700">${a.label}</span>`
                html += `<span style="color:${mutedColor};font-family:monospace;font-size:10px">${new Date(a.t * 1000).toLocaleTimeString()}</span>`
                html += `</div>`
              }
            }
            html += `<div style="color:${mutedColor};margin-bottom:6px;font-family:monospace">${new Date(cursorTime * 1000).toLocaleTimeString()}</div>`

            for (const { i, val, color } of visibleSeries) {
              const s = u.series[i]
              const isClosest = i === closestSeries
              html += `<div style="display:flex;align-items:center;gap:6px;margin-bottom:3px;opacity:${isClosest ? '1' : '0.75'}">`
              html += `<span style="display:inline-block;width:12px;height:${isClosest ? '3' : '2'}px;border-radius:2px;background:${color};flex-shrink:0"></span>`
              html += `<span style="color:${color};font-weight:${isClosest ? '700' : '400'}">${s.label}</span>`
              html += `<span style="color:${fgColor};font-family:monospace;margin-left:4px;font-weight:${isClosest ? '700' : '400'}">${val != null ? fmt(val) : '—'}</span>`
              html += `</div>`
            }

            if (thresholds.length) {
              html += `<div style="margin-top:4px;padding-top:4px;border-top:1px solid ${borderColor}">`
              for (const t of thresholds) {
                if (!t.series.length) continue
                // Find value at cursor time
                let val = t.series[0].v
                for (let i = t.series.length - 1; i >= 0; i--) {
                  if (t.series[i].t <= cursorTime) { val = t.series[i].v; break }
                }
                const isLimit = t.label.toLowerCase().includes('limit')
                const color = isLimit ? '#ef4444' : '#3b82f6'
                const shortLabel = t.label.replace(/:(.*?)$/, ' <span style="color:' + mutedColor + '">$1</span>')
                html += `<div style="display:flex;align-items:center;gap:6px;margin-bottom:3px;opacity:0.85">`
                html += `<span style="display:inline-block;width:12px;border-top:2px dashed ${color};flex-shrink:0"></span>`
                html += `<span style="color:${color}">${shortLabel}</span>`
                html += `<span style="color:${fgColor};font-family:monospace;margin-left:auto">${fmt(val)}</span>`
                html += `</div>`
              }
              html += `</div>`
            }

            // min/max for visible range
            const scaleX = u.scales.x
            const lo = scaleX.min ?? 0
            const hi = scaleX.max ?? 0
            let allVals: number[] = []
            for (let i = 1; i < u.data.length; i++) {
              const arr = u.data[i] as (number | null)[]
              for (let j = 0; j < (u.data[0] as number[]).length; j++) {
                const t = (u.data[0] as number[])[j]
                if (t >= lo && t <= hi && arr[j] != null) allVals.push(arr[j] as number)
              }
            }
            if (allVals.length) {
              const min = Math.min(...allVals)
              const max = Math.max(...allVals)
              html += `<div style="color:${mutedColor};margin-top:4px;padding-top:4px;border-top:1px solid ${borderColor};font-family:monospace">min: ${fmt(min)} / max: ${fmt(max)}</div>`
            }

            html += '</div>'

            // Convert cursor coords (relative to u.over) into coords relative to
            // the tooltip's positioning parent (the .relative wrapper div).
            tooltip.style.display = 'block'
            tooltip.innerHTML = html

            const overRect = u.over.getBoundingClientRect()
            const parentRect = tooltip.parentElement!.getBoundingClientRect()
            const absX = overRect.left - parentRect.left + x
            const absY = overRect.top - parentRect.top + y

            const tipW = tooltip.offsetWidth
            const tipLeft = absX + tipW + 16 > parentRect.width ? absX - tipW - 8 : absX + 16
            tooltip.style.left = `${tipLeft}px`
            tooltip.style.top = `${Math.max(0, absY - 10)}px`
          },
        ],
        setScale: [
          (u, key) => {
            if (key === 'x' && !isProgrammaticScale) {
              const { min, max } = u.scales.x
              if (min != null && max != null) {
                onzoom?.({ min, max })
              }
            }
          },
        ],
        ready: [
          (u) => {
            u.over.addEventListener('dblclick', () => {
              onzoom?.(null)
              // Also reset locally — the $effect will cover other charts
              const data = u.data[0] as number[]
              if (data.length) {
                isProgrammaticScale = true
                u.setScale('x', { min: data[0], max: data[data.length - 1] })
                isProgrammaticScale = false
              }
            })
            u.over.addEventListener('mouseleave', () => {
              if (tooltip) tooltip.style.display = 'none'
            })
          },
        ],
        drawSeries: [
          (u) => {
            if (!thresholds.length) return
            const ctx = u.ctx
            // u.bbox is in canvas (physical) pixels; valToPos(..., true) also returns canvas pixels
            const left = u.bbox.left
            const right = u.bbox.left + u.bbox.width
            const top = u.bbox.top
            const bottom = u.bbox.top + u.bbox.height
            const xMax = u.scales.x.max ?? 0

            ctx.save()
            ctx.beginPath()
            ctx.rect(left, top, u.bbox.width, u.bbox.height)
            ctx.clip()
            ctx.lineWidth = devicePixelRatio
            ctx.setLineDash([4 * devicePixelRatio, 4 * devicePixelRatio])

            for (const t of thresholds) {
              if (!t.series.length) continue
              // Find the value nearest to the right edge of the visible range
              let val: number | null = null
              for (let i = t.series.length - 1; i >= 0; i--) {
                if (t.series[i].t <= xMax) { val = t.series[i].v; break }
              }
              if (val == null) val = t.series[0].v

              const y = Math.round(u.valToPos(val, 'y', true))
              const isLimit = t.label.toLowerCase().includes('limit')
              ctx.strokeStyle = isLimit ? '#ef4444' : '#3b82f6'
              ctx.beginPath()
              ctx.moveTo(left, y)
              ctx.lineTo(right, y)
              ctx.stroke()
            }
            ctx.restore()
          },
        ],
        drawAxes: [
          (u) => {
            const ann = sortedAnnotations
            if (!ann.length) return
            const ctx = u.ctx
            const top = u.bbox.top
            const bottom = u.bbox.top + u.bbox.height
            const xMin = u.scales.x.min ?? 0
            const xMax = u.scales.x.max ?? 0

            // Binary search for first annotation in visible range
            let lo = 0, hi = ann.length
            while (lo < hi) {
              const mid = (lo + hi) >> 1
              if (ann[mid].t < xMin) lo = mid + 1
              else hi = mid
            }

            const triSize = 4 * devicePixelRatio
            ctx.save()
            ctx.lineWidth = devicePixelRatio
            ctx.setLineDash([])
            for (let i = lo; i < ann.length && ann[i].t <= xMax; i++) {
              const x = Math.round(u.valToPos(ann[i].t, 'x', true))
              ctx.strokeStyle = ANNOTATION_COLORS[ann[i].severity] ?? '#3b82f6'
              ctx.globalAlpha = 0.6
              ctx.beginPath()
              ctx.moveTo(x, top)
              ctx.lineTo(x, bottom)
              ctx.stroke()
              ctx.globalAlpha = 1
              ctx.fillStyle = ANNOTATION_COLORS[ann[i].severity] ?? '#3b82f6'
              ctx.beginPath()
              ctx.moveTo(x - triSize, bottom)
              ctx.lineTo(x + triSize, bottom)
              ctx.lineTo(x, bottom - triSize * 1.5)
              ctx.closePath()
              ctx.fill()
            }
            ctx.restore()
          },
        ],
      },
    }

    chart = new uPlot(opts, toColumnar(initialSeries), container)

    // Make drag-selection rectangle visible using the accent color
    const accentColor = style.getPropertyValue('--color-accent').trim() || '#3b82f6'
    const selectEl = chart.root?.querySelector('.u-select') as HTMLElement | null
    if (selectEl) {
      selectEl.style.background = `color-mix(in srgb, ${accentColor} 20%, transparent)`
      selectEl.style.borderLeft = `2px solid ${accentColor}`
      selectEl.style.borderRight = `2px solid ${accentColor}`
    }

    const ro = new ResizeObserver(() => {
      if (chart && container) {
        chart.setSize({ width: container.clientWidth, height })
      }
    })
    ro.observe(container)

    return () => {
      ro.disconnect()
      chart?.destroy()
      chart = null
    }
  })

  // Apply externally-driven zoom range without re-emitting onzoom
  $effect(() => {
    const range = zoomRange
    untrack(() => {
      if (!chart) return
      isProgrammaticScale = true
      if (range) {
        chart.setScale('x', { min: range.min, max: range.max })
      } else {
        const data = chart.data[0] as number[]
        if (data?.length) chart.setScale('x', { min: data[0], max: data[data.length - 1] })
      }
      isProgrammaticScale = false
    })
  })

  $effect(() => {
    // Reactive data update — do NOT recreate chart
    const newData = toColumnar(series)
    untrack(() => {
      if (!chart) return
      // Pad missing value arrays to match chart's series count (x + N series).
      // Mismatch happens when switching resources and series resets to [] before
      // the chart is recreated — uPlot crashes on undefined data[i].
      const expected = chart.series.length
      while (newData.length < expected) {
        ;(newData as unknown[]).push(new Float64Array())
      }
      chart.setData(newData)
    })
  })
</script>

<div class="relative">
  <div class="text-sm font-medium text-fg mb-1">{title}</div>
  {#if loading}
    <div class="animate-pulse bg-surface rounded" style="height:{height}px"></div>
  {:else}
    <div class="relative">
      <div bind:this={container}></div>
      <div
        bind:this={tooltip}
        class="pointer-events-none absolute z-10 hidden rounded bg-surface border border-border px-2 py-1 shadow text-xs"
      ></div>
    </div>
    {#if series.length > 0}
      <div class="flex flex-wrap gap-2 mt-1">
        {#each series as s, i}
          {@const label = seriesLabel(s, i)}
          {@const color = COLORS[i % COLORS.length]}
          <button
            class="flex items-center gap-1 text-xs transition-opacity"
            style="opacity: {legendToggles[i] === false ? '0.4' : '1'}"
            onclick={() => {
              legendToggles[i] = legendToggles[i] === false ? true : false
              if (chart) chart.setSeries(i + 1, { show: legendToggles[i] !== false })
            }}
          >
            <span class="inline-block w-3 h-0.5 rounded" style="background:{color}"></span>
            {label}
          </button>
        {/each}
      </div>
    {/if}
  {/if}
</div>
