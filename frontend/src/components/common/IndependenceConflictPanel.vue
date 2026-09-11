<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { GitBranch } from 'lucide-vue-next'
import { useIndependenceConflictStore } from '../../stores/independence-conflict'
import { errorMessage } from '../../api/client'
import { conflictResolutionActionLabels, conflictResolutionActions, conflictResolutionStateLabels } from '../../types/enums/conflict-resolution'
import type { ConflictResolutionAction } from '../../types/enums/conflict-resolution'
import type { IndependenceConflict } from '../../types/independence-conflict'

const props = defineProps<{ evaluationId: number; evaluationState: string; canReview: boolean }>()
const store = useIndependenceConflictStore()

const forms = reactive<Record<number, { action: ConflictResolutionAction; note: string; submitting: boolean }>>({})
const items = computed(() => store.items)
const priors = computed(() => store.priorResolutions)
const pendingCount = computed(() => store.pendingCount)
const canAct = computed(() => props.canReview && props.evaluationState === 'completed')

watch(() => props.evaluationId, async (id) => {
  store.reset()
  if (!id) return
  try { await store.load(id) } catch (error) { ElMessage.error(errorMessage(error)) }
}, { immediate: true })

function formFor(id: number) {
  if (!forms[id]) forms[id] = { action: 'accept', note: '', submitting: false }
  return forms[id]
}

async function submit(item: IndependenceConflict) {
  const form = formFor(item.id)
  if (form.note.trim().length < 3) { ElMessage.warning('请填写至少 3 个字符的复核理由'); return }
  form.submitting = true
  try {
    await store.resolve(item.id, { action: form.action, note: form.note.trim() })
    ElMessage.success('冲突复核结论已记录')
  } catch (error) { ElMessage.error(errorMessage(error)) } finally { form.submitting = false }
}

const actionLabel = (action?: string) => action ? conflictResolutionActionLabels[action as ConflictResolutionAction] ?? action : ''
const ignoredLabels = (item: IndependenceConflict) => item.ignored_safeguard_ids.map((id) => `#${id}`).join('、')
</script>

<template>
  <section class="conflict-panel" aria-label="独立性冲突复核">
    <div class="section-heading">
      <div><p class="eyebrow">INDEPENDENCE CONFLICT REVIEW</p><h2><GitBranch :size="16" />独立性冲突复核</h2></div>
      <span>{{ items.length ? `${pendingCount} 组待复核 · 共 ${items.length} 组` : '无同键冲突' }}</span>
    </div>
    <div v-if="!items.length" class="empty-inline">本次评估没有同一独立性键的去重冲突，可直接复核确认。</div>
    <article v-for="item in items" :key="item.id" class="conflict-group" :data-state="item.resolution_state">
      <header>
        <strong>{{ item.independence_key }}</strong>
        <span class="state-label" :class="item.resolution_state">{{ conflictResolutionStateLabels[item.resolution_state] }}</span>
      </header>
      <p class="dedup-basis">保留保护层 #{{ item.kept_safeguard_id }}，忽略 {{ ignoredLabels(item) }}。依据：{{ item.dedup_reason }}</p>
      <div v-if="item.resolution_state !== 'pending'" class="resolution-summary">
        <div><strong>{{ actionLabel(item.resolution_action) }}</strong><span>{{ item.resolved_by_name }}<template v-if="item.resolved_at"> · {{ new Date(item.resolved_at).toLocaleString('zh-CN') }}</template></span></div>
        <p>理由：{{ item.resolution_note }}</p>
      </div>
      <div v-else-if="canAct" class="resolution-form">
        <el-select v-model="formFor(item.id).action" aria-label="处理方式">
          <el-option v-for="action in conflictResolutionActions" :key="action" :value="action" :label="conflictResolutionActionLabels[action]" />
        </el-select>
        <el-input v-model="formFor(item.id).note" type="textarea" :rows="2" maxlength="1000" placeholder="填写复核理由（必填，随审计保留）" />
        <el-button type="primary" :loading="formFor(item.id).submitting" @click="submit(item)">提交复核结论</el-button>
      </div>
      <p v-else class="conflict-waiting">待评估进入待确认状态后由安全复核员逐组处理。</p>
    </article>
    <div v-if="priors.length" class="prior-resolutions">
      <h3>历史处理结果</h3>
      <p>同场景以往评估（含已作废）的复核结论，重算后保留，仅供当前复核参考。</p>
      <ul>
        <li v-for="prior in priors" :key="`${prior.evaluation_id}-${prior.id}`">
          评估 #{{ prior.evaluation_id }} · {{ prior.independence_key }} · {{ actionLabel(prior.resolution_action) }} · {{ prior.resolution_note }}
        </li>
      </ul>
    </div>
  </section>
</template>
