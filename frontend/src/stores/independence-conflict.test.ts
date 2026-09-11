import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { IndependenceConflict, IndependenceConflictList } from '../types/independence-conflict'

const listMock = vi.fn<(id: number) => Promise<IndependenceConflictList>>()
const resolveMock = vi.fn<(evaluationId: number, conflictId: number, input: { action: string; note: string }) => Promise<IndependenceConflict>>()

vi.mock('../api/independence-conflict', () => ({
  listIndependenceConflicts: (id: number) => listMock(id),
  resolveIndependenceConflict: (evaluationId: number, conflictId: number, input: { action: string; note: string }) => resolveMock(evaluationId, conflictId, input),
}))

import { useIndependenceConflictStore } from './independence-conflict'

const pendingGroup: IndependenceConflict = {
  id: 11, evaluation_id: 7, scenario_id: 3, independence_key: 'SIS-A',
  kept_safeguard_id: 2, ignored_safeguard_ids: [3], dedup_reason: 'same independence_key',
  resolution_state: 'pending', created_at: '2026-09-11T00:00:00Z',
}
const resolvedGroup: IndependenceConflict = {
  ...pendingGroup, id: 12, independence_key: 'PSV-B', kept_safeguard_id: 4, ignored_safeguard_ids: [5],
  resolution_state: 'accepted', resolution_action: 'accept', resolution_note: '接受去重',
  resolved_by: 20, resolved_by_name: 'reviewer', resolved_at: '2026-09-11T01:00:00Z',
}

describe('independence-conflict store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listMock.mockReset()
    resolveMock.mockReset()
  })

  it('loads conflict groups with pending count and retained prior resolutions', async () => {
    listMock.mockResolvedValue({ items: [pendingGroup, resolvedGroup], pending_count: 1, prior_resolutions: [resolvedGroup] })
    const store = useIndependenceConflictStore()
    await store.load(7)
    expect(listMock).toHaveBeenCalledWith(7)
    expect(store.items).toHaveLength(2)
    expect(store.pendingCount).toBe(1)
    expect(store.priorResolutions).toHaveLength(1)
    expect(store.priorResolutions[0].resolution_note).toBe('接受去重')
  })

  it('resolves a group and recomputes the pending count', async () => {
    listMock.mockResolvedValue({ items: [pendingGroup], pending_count: 1, prior_resolutions: [] })
    resolveMock.mockResolvedValue({ ...pendingGroup, resolution_state: 'split', resolution_action: 'split', resolution_note: '拆分保护层' })
    const store = useIndependenceConflictStore()
    await store.load(7)
    const updated = await store.resolve(11, { action: 'split', note: '拆分保护层' })
    expect(resolveMock).toHaveBeenCalledWith(7, 11, { action: 'split', note: '拆分保护层' })
    expect(updated.resolution_state).toBe('split')
    expect(store.items[0].resolution_state).toBe('split')
    expect(store.pendingCount).toBe(0)
  })

  it('clears stale groups when reset before loading another evaluation', async () => {
    listMock.mockResolvedValue({ items: [pendingGroup], pending_count: 1, prior_resolutions: [] })
    const store = useIndependenceConflictStore()
    await store.load(7)
    store.reset()
    expect(store.items).toHaveLength(0)
    expect(store.pendingCount).toBe(0)
    await expect(store.resolve(11, { action: 'accept', note: '接受去重' })).rejects.toThrow()
  })
})
