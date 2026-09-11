import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as conflictApi from '../api/independence-conflict'
import type { IndependenceConflict, ResolveConflictInput } from '../types/independence-conflict'

export const useIndependenceConflictStore = defineStore('independence-conflicts', () => {
  const evaluationId = ref<number>()
  const items = ref<IndependenceConflict[]>([])
  const priorResolutions = ref<IndependenceConflict[]>([])
  const pendingCount = ref(0)
  const loading = ref(false)

  function reset() {
    evaluationId.value = undefined
    items.value = []
    priorResolutions.value = []
    pendingCount.value = 0
  }

  async function load(id: number) {
    loading.value = true
    try {
      const result = await conflictApi.listIndependenceConflicts(id)
      evaluationId.value = id
      items.value = result.items
      priorResolutions.value = result.prior_resolutions ?? []
      pendingCount.value = result.pending_count
    } finally {
      loading.value = false
    }
  }

  async function resolve(id: number, input: ResolveConflictInput) {
    if (evaluationId.value === undefined) throw new Error('尚未加载冲突列表')
    const updated = await conflictApi.resolveIndependenceConflict(evaluationId.value, id, input)
    const index = items.value.findIndex((item) => item.id === id)
    if (index >= 0) items.value[index] = updated
    pendingCount.value = items.value.filter((item) => item.resolution_state === 'pending').length
    return updated
  }

  return { evaluationId, items, priorResolutions, pendingCount, loading, reset, load, resolve }
})
