import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import type { IndependenceConflict, IndependenceConflictList } from '../../types/independence-conflict'

const listMock = vi.fn<(id: number) => Promise<IndependenceConflictList>>()
const resolveMock = vi.fn<(evaluationId: number, conflictId: number, input: { action: string; note: string }) => Promise<IndependenceConflict>>()

vi.mock('../../api/independence-conflict', () => ({
  listIndependenceConflicts: (id: number) => listMock(id),
  resolveIndependenceConflict: (evaluationId: number, conflictId: number, input: { action: string; note: string }) => resolveMock(evaluationId, conflictId, input),
}))

import IndependenceConflictPanel from './IndependenceConflictPanel.vue'

const stubs = {
  ElSelect: {
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: `<select :value="modelValue" @change="$emit('update:modelValue', $event.target.value)"><slot /></select>`,
  },
  ElOption: { props: ['value', 'label'], template: `<option :value="value">{{ label }}</option>` },
  ElInput: {
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: `<textarea :value="modelValue" @input="$emit('update:modelValue', $event.target.value)" />`,
  },
  ElButton: { props: ['loading'], emits: ['click'], template: `<button type="button" @click="$emit('click')"><slot /></button>` },
}

const pendingGroup: IndependenceConflict = {
  id: 11, evaluation_id: 7, scenario_id: 3, independence_key: 'SIS-A',
  kept_safeguard_id: 2, ignored_safeguard_ids: [3, 4], dedup_reason: 'same independence_key on one path',
  resolution_state: 'pending', created_at: '2026-09-11T00:00:00Z',
}
const priorResolved: IndependenceConflict = {
  ...pendingGroup, id: 5, evaluation_id: 6,
  resolution_state: 'accepted', resolution_action: 'accept', resolution_note: '历史评估已接受去重',
  resolved_by: 20, resolved_by_name: 'reviewer', resolved_at: '2026-09-01T00:00:00Z',
}

function mountPanel(props: { evaluationState?: string; canReview?: boolean } = {}) {
  return mount(IndependenceConflictPanel, {
    props: { evaluationId: 7, evaluationState: props.evaluationState ?? 'completed', canReview: props.canReview ?? true },
    global: { stubs },
  })
}

describe('IndependenceConflictPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listMock.mockReset()
    resolveMock.mockReset()
  })

  it('lists conflict groups with kept and ignored basis', async () => {
    listMock.mockResolvedValue({ items: [pendingGroup], pending_count: 1, prior_resolutions: [] })
    const wrapper = mountPanel()
    await flushPromises()
    expect(listMock).toHaveBeenCalledWith(7)
    expect(wrapper.text()).toContain('SIS-A')
    expect(wrapper.text()).toContain('保留保护层 #2，忽略 #3、#4')
    expect(wrapper.text()).toContain('same independence_key on one path')
    expect(wrapper.text()).toContain('待复核')
  })

  it('lets a reviewer resolve a pending group with a note', async () => {
    listMock.mockResolvedValue({ items: [pendingGroup], pending_count: 1, prior_resolutions: [] })
    resolveMock.mockImplementation(async (_evaluationId, _conflictId, input) => ({
      ...pendingGroup, resolution_state: 'accepted', resolution_action: input.action as IndependenceConflict['resolution_action'],
      resolution_note: input.note, resolved_by_name: 'reviewer', resolved_at: '2026-09-11T02:00:00Z',
    }))
    const wrapper = mountPanel()
    await flushPromises()
    await wrapper.find('textarea').setValue('同键报警与 SIS 共用最终元件')
    await wrapper.find('button').trigger('click')
    await flushPromises()
    expect(resolveMock).toHaveBeenCalledWith(7, 11, { action: 'accept', note: '同键报警与 SIS 共用最终元件' })
    expect(wrapper.text()).toContain('已接受去重')
    expect(wrapper.text()).toContain('理由：同键报警与 SIS 共用最终元件')
    expect(wrapper.find('textarea').exists()).toBe(false)
  })

  it('blocks submission without a meaningful note', async () => {
    listMock.mockResolvedValue({ items: [pendingGroup], pending_count: 1, prior_resolutions: [] })
    const wrapper = mountPanel()
    await flushPromises()
    await wrapper.find('textarea').setValue('no')
    await wrapper.find('button').trigger('click')
    await flushPromises()
    expect(resolveMock).not.toHaveBeenCalled()
  })

  it('hides the form for auditors and voided evaluations', async () => {
    listMock.mockResolvedValue({ items: [pendingGroup], pending_count: 1, prior_resolutions: [] })
    const auditor = mountPanel({ canReview: false })
    await flushPromises()
    expect(auditor.find('textarea').exists()).toBe(false)
    const voided = mountPanel({ evaluationState: 'voided' })
    await flushPromises()
    expect(voided.find('textarea').exists()).toBe(false)
  })

  it('shows retained prior resolutions from voided or recalculated evaluations', async () => {
    listMock.mockResolvedValue({ items: [pendingGroup], pending_count: 1, prior_resolutions: [priorResolved] })
    const wrapper = mountPanel()
    await flushPromises()
    expect(wrapper.text()).toContain('历史处理结果')
    expect(wrapper.text()).toContain('评估 #6 · SIS-A · 接受去重 · 历史评估已接受去重')
  })
})
