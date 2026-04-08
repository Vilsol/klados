import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'

const { mockGetColumnPrefs, mockGetCompactRows } = vi.hoisted(() => ({
  mockGetColumnPrefs: vi.fn().mockResolvedValue(null),
  mockGetCompactRows: vi.fn().mockResolvedValue(false),
}))

vi.mock('../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js', () => ({
  GetColumnPrefs: mockGetColumnPrefs,
  SetColumnPrefs: vi.fn(),
  DeleteColumnPrefs: vi.fn(),
  GetCompactRows: mockGetCompactRows,
  SetCompactRows: vi.fn(),
}))

vi.mock('../registry/index.js', () => ({
  descriptorRegistry: {
    get: vi.fn().mockReturnValue({
      columns: [],
      overviewFields: [],
      detailPanels: [],
      actions: [],
    }),
  },
}))

import { columnStore } from '$lib/stores/columns.svelte'
import ColumnMenu from '$lib/components/ColumnMenu.svelte'

const col = (name: string) => ({ name, expr: `metadata.${name.toLowerCase()}`, renderType: 'text' as const })

const fiveColumns = [
  { col: col('Name'), visible: true },
  { col: col('Namespace'), visible: true },
  { col: col('Ready'), visible: true },
  { col: col('Age'), visible: true },
  { col: col('Status'), visible: false },
]

describe('ColumnMenu', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    columnStore.visibleColumns = fiveColumns.filter((e) => e.visible).map((e) => e.col)
    columnStore.allColumns = fiveColumns
    columnStore.compact = false
  })

  it('renders all columns', () => {
    render(ColumnMenu, { props: { gvr: 'core.v1.pods' } })
    expect(screen.getByText('Name')).toBeTruthy()
    expect(screen.getByText('Namespace')).toBeTruthy()
    expect(screen.getByText('Ready')).toBeTruthy()
    expect(screen.getByText('Age')).toBeTruthy()
    expect(screen.getByText('Status')).toBeTruthy()
  })

  it('Name column checkbox is disabled and checked', () => {
    render(ColumnMenu, { props: { gvr: 'core.v1.pods' } })
    const checkboxes = screen.getAllByRole('checkbox')
    // Name is the first visible entry → first checkbox
    expect((checkboxes[0] as HTMLInputElement).disabled).toBe(true)
    expect((checkboxes[0] as HTMLInputElement).checked).toBe(true)
  })

  it('toggling visibility calls setColumnVisible', async () => {
    const spy = vi.spyOn(columnStore, 'setColumnVisible')
    render(ColumnMenu, { props: { gvr: 'core.v1.pods' } })
    // Namespace is the second visible entry → second checkbox (idx 1)
    const checkboxes = screen.getAllByRole('checkbox')
    await fireEvent.click(checkboxes[1])
    expect(spy).toHaveBeenCalledWith('Namespace', expect.any(Boolean))
  })

  it('up button is disabled for the second visible column (cannot move above Name)', () => {
    render(ColumnMenu, { props: { gvr: 'core.v1.pods' } })
    const upBtn = screen.getByRole('button', { name: 'Move Namespace up' })
    expect((upBtn as HTMLButtonElement).disabled).toBe(true)
  })

  it('down button is disabled for the last visible column', () => {
    render(ColumnMenu, { props: { gvr: 'core.v1.pods' } })
    // Age is last visible column
    const downBtn = screen.getByRole('button', { name: 'Move Age down' })
    expect((downBtn as HTMLButtonElement).disabled).toBe(true)
  })

  it('reset button calls columnStore.reset', async () => {
    const spy = vi.spyOn(columnStore, 'reset')
    render(ColumnMenu, { props: { gvr: 'core.v1.pods' } })
    const resetBtn = screen.getByRole('button', { name: /^reset$/i })
    await fireEvent.click(resetBtn)
    expect(spy).toHaveBeenCalled()
  })

  it('sparkline section is hidden when GVR is not in sparklineGvrs', () => {
    render(ColumnMenu, {
      props: {
        gvr: 'core.v1.configmaps',
        sparklineGvrs: ['core.v1.pods'],
        sparklineColumns: [],
      },
    })
    expect(screen.queryByText('Sparklines')).toBeNull()
    expect(screen.queryByText('CPU')).toBeNull()
    expect(screen.queryByText('Memory')).toBeNull()
  })

  it('sparkline section appears when GVR is in sparklineGvrs', () => {
    render(ColumnMenu, {
      props: {
        gvr: 'core.v1.pods',
        sparklineGvrs: ['core.v1.pods'],
        sparklineColumns: [],
      },
    })
    expect(screen.getByText('Sparklines')).toBeTruthy()
    expect(screen.getByText('CPU')).toBeTruthy()
    expect(screen.getByText('Memory')).toBeTruthy()
  })
})
