<script lang="ts">
  import MetricsChart from '../components/charts/MetricsChart.svelte'
  import type { TimeSeries } from '../components/charts/types'

  let series: TimeSeries[] = $state([
    { labels: { container: 'c0' }, points: [{ t: 1000, v: 0.1 }, { t: 1015, v: 0.2 }] },
  ])

  function updateData() {
    series = [
      { labels: { container: 'c0' }, points: [{ t: 1000, v: 0.1 }, { t: 1015, v: 0.2 }, { t: 1030, v: 0.3 }] },
    ]
  }
</script>

<MetricsChart title="CPU" unit="cores" {series} />
<button data-testid="update-data" onclick={updateData}>update</button>
