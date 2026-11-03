import { useState } from 'react'
import {
  MAX_CONDITIONAL_ORDERS,
  MENTALITY_ATTACKING,
  ORDER_ACTION_MENTALITY,
  ORDER_ACTION_SUBSTITUTION,
  ORDER_CONDITION_LOSING,
  ORDER_MAX_FROM_MINUTE,
  ORDER_MIN_FROM_MINUTE,
  type Lineup,
  type LineupOrder,
} from '../../api/types'

// A conditional order being edited: the minute and the players start empty ('') until picked.
export interface OrderDraft {
  fromMinute: number | ''
  condition: number
  action: number
  mentality: number
  playerOutId: number | ''
  playerInId: number | ''
}

function initialDrafts(lineup: Lineup): OrderDraft[] {
  return (lineup.orders ?? []).map((o) => ({
    fromMinute: o.fromMinute,
    condition: o.condition,
    action: o.action,
    mentality: o.mentality,
    playerOutId: o.playerOutId ?? '',
    playerInId: o.playerInId ?? '',
  }))
}

export function isValidFromMinute(minute: number | ''): boolean {
  return minute !== '' && minute >= ORDER_MIN_FROM_MINUTE && minute <= ORDER_MAX_FROM_MINUTE
}

// Drives the conditional-orders section of MatchLineupForm (openspec/specs/match-simulation
// "Conditional match orders"), the same way useSubstitutionDrafts drives the planned substitutions.
// An order with a from-minute outside 1-89 makes the whole set invalid (the form won't save it);
// a substitution order whose players are missing or no longer starter/bench is dropped on save,
// like an incomplete substitution.
export function useOrderDrafts(lineup: Lineup, usedPlayerIds: Set<number | ''>) {
  const [orderDrafts, setOrderDrafts] = useState<OrderDraft[]>(() => initialDrafts(lineup))

  const handleAddOrder = () => {
    if (orderDrafts.length >= MAX_CONDITIONAL_ORDERS) return
    setOrderDrafts((prev) => [
      ...prev,
      { fromMinute: '', condition: ORDER_CONDITION_LOSING, action: ORDER_ACTION_MENTALITY, mentality: MENTALITY_ATTACKING, playerOutId: '', playerInId: '' },
    ])
  }
  const handleRemoveOrder = (index: number) => setOrderDrafts((prev) => prev.filter((_, i) => i !== index))
  const handleOrderChange = (index: number, field: keyof OrderDraft, value: number | '') =>
    setOrderDrafts((prev) => prev.map((o, i) => (i === index ? { ...o, [field]: value } : o)))
  const resetOrderDrafts = () => setOrderDrafts([])

  const ordersValid = orderDrafts.every((o) => isValidFromMinute(o.fromMinute))
  const validOrders: LineupOrder[] = orderDrafts.flatMap((o): LineupOrder[] => {
    if (!isValidFromMinute(o.fromMinute)) return []
    const fromMinute = o.fromMinute as number
    if (o.action === ORDER_ACTION_SUBSTITUTION) {
      if (o.playerOutId === '' || o.playerInId === '' || !usedPlayerIds.has(o.playerOutId) || usedPlayerIds.has(o.playerInId)) return []
      return [{ fromMinute, condition: o.condition, action: o.action, mentality: 0, playerOutId: o.playerOutId, playerInId: o.playerInId }]
    }
    return [{ fromMinute, condition: o.condition, action: ORDER_ACTION_MENTALITY, mentality: o.mentality }]
  })

  return { orderDrafts, handleAddOrder, handleRemoveOrder, handleOrderChange, resetOrderDrafts, ordersValid, validOrders }
}
