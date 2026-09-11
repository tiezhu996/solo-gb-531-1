export const conflictResolutionStates = ['pending', 'accepted', 'split', 'evidenced'] as const
export type ConflictResolutionState = (typeof conflictResolutionStates)[number]

export const conflictResolutionStateLabels: Record<ConflictResolutionState, string> = {
  pending: '待复核',
  accepted: '已接受去重',
  split: '拆分保护层',
  evidenced: '已补充证据',
}

export const conflictResolutionActions = ['accept', 'split', 'evidence'] as const
export type ConflictResolutionAction = (typeof conflictResolutionActions)[number]

export const conflictResolutionActionLabels: Record<ConflictResolutionAction, string> = {
  accept: '接受去重',
  split: '拆分保护层',
  evidence: '补充证据',
}
