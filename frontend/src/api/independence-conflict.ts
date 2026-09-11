import { api, json } from './client'
import type { IndependenceConflict, IndependenceConflictList, ResolveConflictInput } from '../types/independence-conflict'

export const listIndependenceConflicts = (evaluationId: number) =>
  api<IndependenceConflictList>(`/coverage-evaluations/${evaluationId}/independence-conflicts`)

export const resolveIndependenceConflict = (evaluationId: number, conflictId: number, input: ResolveConflictInput) =>
  api<IndependenceConflict>(`/coverage-evaluations/${evaluationId}/independence-conflicts/${conflictId}/resolve`, json('POST', input))
