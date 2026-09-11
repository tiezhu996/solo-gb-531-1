import type { ConflictResolutionAction, ConflictResolutionState } from './enums/conflict-resolution'

export interface IndependenceConflict {
  id: number
  evaluation_id: number
  scenario_id: number
  independence_key: string
  kept_safeguard_id: number
  ignored_safeguard_ids: number[]
  dedup_reason: string
  resolution_state: ConflictResolutionState
  resolution_action?: ConflictResolutionAction
  resolution_note?: string
  resolved_by?: number
  resolved_by_name?: string
  resolved_at?: string
  created_at: string
}

export interface IndependenceConflictList {
  items: IndependenceConflict[]
  pending_count: number
  prior_resolutions: IndependenceConflict[]
}

export interface ResolveConflictInput {
  action: ConflictResolutionAction
  note: string
}
