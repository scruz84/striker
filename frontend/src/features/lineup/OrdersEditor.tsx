import Button from '@mui/material/Button'
import FormControl from '@mui/material/FormControl'
import IconButton from '@mui/material/IconButton'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Select from '@mui/material/Select'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined'
import { useTranslation } from 'react-i18next'
import {
  MAX_CONDITIONAL_ORDERS,
  MENTALITY_ATTACKING,
  MENTALITY_BALANCED,
  MENTALITY_DEFENSIVE,
  MENTALITY_KEYS,
  ORDER_ACTION_MENTALITY,
  ORDER_ACTION_SUBSTITUTION,
  ORDER_CONDITION_KEYS,
  ORDER_MAX_FROM_MINUTE,
  ORDER_MIN_FROM_MINUTE,
  type Player,
} from '../../api/types'
import { NumberField } from '../../components/NumberField'
import { isValidFromMinute, type OrderDraft } from './useOrderDrafts'

export interface OrdersEditorProps {
  starters: Player[]
  benchPlayers: Player[]
  unavailability: Map<number, unknown>
  drafts: OrderDraft[]
  onAdd: () => void
  onRemove: (index: number) => void
  onChange: (index: number, field: keyof OrderDraft, value: number | '') => void
}

// The conditional-orders section of MatchLineupForm (openspec/specs/match-simulation "Conditional
// match orders"): up to MAX_CONDITIONAL_ORDERS plans of the form "from minute M, if <score>, then
// <change mentality | substitute>", which the engine follows minute by minute.
export function OrdersEditor({ starters, benchPlayers, unavailability, drafts, onAdd, onRemove, onChange }: OrdersEditorProps) {
  const { t } = useTranslation()

  return (
    <Paper variant="outlined" sx={{ p: { xs: 1.5, sm: 2 }, mt: { xs: 2, sm: 3 } }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('lineup.orders.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {t('lineup.orders.description', { max: MAX_CONDITIONAL_ORDERS })}
      </Typography>
      <Stack spacing={1.5}>
        {drafts.map((draft, index) => {
          const minuteInvalid = draft.fromMinute !== '' && !isValidFromMinute(draft.fromMinute)
          return (
            <Paper key={index} variant="outlined" sx={{ p: 1.5 }}>
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
                <Typography variant="subtitle2">{t('lineup.orders.orderN', { n: index + 1 })}</Typography>
                <IconButton onClick={() => onRemove(index)} aria-label={t('lineup.orders.remove')} edge="end">
                  <DeleteOutlinedIcon />
                </IconButton>
              </Stack>
              <Stack spacing={1.5}>
                <NumberField
                  label={t('lineup.orders.fromMinute')}
                  value={draft.fromMinute === '' ? '' : String(draft.fromMinute)}
                  onValueChange={(digits) => onChange(index, 'fromMinute', digits === '' ? '' : Number(digits))}
                  min={ORDER_MIN_FROM_MINUTE}
                  max={ORDER_MAX_FROM_MINUTE}
                  suffix="'"
                  error={minuteInvalid}
                  helperText={minuteInvalid ? t('lineup.orders.fromMinuteRange', { min: ORDER_MIN_FROM_MINUTE, max: ORDER_MAX_FROM_MINUTE }) : undefined}
                  fullWidth
                />
                <FormControl fullWidth>
                  <InputLabel id={`order-condition-${index}`}>{t('lineup.orders.condition')}</InputLabel>
                  <Select
                    labelId={`order-condition-${index}`}
                    label={t('lineup.orders.condition')}
                    value={draft.condition}
                    onChange={(e) => onChange(index, 'condition', Number(e.target.value))}
                  >
                    {Object.entries(ORDER_CONDITION_KEYS).map(([value, key]) => (
                      <MenuItem key={value} value={Number(value)}>
                        {t(`lineup.orders.conditions.${key}`)}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
                <FormControl fullWidth>
                  <InputLabel id={`order-action-${index}`}>{t('lineup.orders.action')}</InputLabel>
                  <Select
                    labelId={`order-action-${index}`}
                    label={t('lineup.orders.action')}
                    value={draft.action}
                    onChange={(e) => onChange(index, 'action', Number(e.target.value))}
                  >
                    <MenuItem value={ORDER_ACTION_MENTALITY}>{t('lineup.orders.actions.mentality')}</MenuItem>
                    <MenuItem value={ORDER_ACTION_SUBSTITUTION}>{t('lineup.orders.actions.substitution')}</MenuItem>
                  </Select>
                </FormControl>
                {draft.action === ORDER_ACTION_MENTALITY ? (
                  <FormControl fullWidth>
                    <InputLabel id={`order-mentality-${index}`}>{t('lineup.mentality.label')}</InputLabel>
                    <Select
                      labelId={`order-mentality-${index}`}
                      label={t('lineup.mentality.label')}
                      value={draft.mentality}
                      onChange={(e) => onChange(index, 'mentality', Number(e.target.value))}
                    >
                      {[MENTALITY_DEFENSIVE, MENTALITY_BALANCED, MENTALITY_ATTACKING].map((m) => (
                        <MenuItem key={m} value={m}>
                          {t(`lineup.mentality.${MENTALITY_KEYS[m]}`)}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                ) : (
                  <>
                    <FormControl fullWidth>
                      <InputLabel id={`order-out-${index}`}>{t('lineup.substitutions.playerOut')}</InputLabel>
                      <Select
                        labelId={`order-out-${index}`}
                        label={t('lineup.substitutions.playerOut')}
                        value={draft.playerOutId}
                        onChange={(e) => onChange(index, 'playerOutId', e.target.value as number)}
                      >
                        {starters.map((p) => (
                          <MenuItem key={p.id} value={p.id}>
                            {p.firstName} {p.lastName}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                    <FormControl fullWidth>
                      <InputLabel id={`order-in-${index}`}>{t('lineup.substitutions.playerIn')}</InputLabel>
                      <Select
                        labelId={`order-in-${index}`}
                        label={t('lineup.substitutions.playerIn')}
                        value={draft.playerInId}
                        onChange={(e) => onChange(index, 'playerInId', e.target.value as number)}
                      >
                        {benchPlayers
                          .filter((p) => !unavailability.has(p.id))
                          .map((p) => (
                            <MenuItem key={p.id} value={p.id}>
                              {p.firstName} {p.lastName}
                            </MenuItem>
                          ))}
                      </Select>
                    </FormControl>
                  </>
                )}
              </Stack>
            </Paper>
          )
        })}
      </Stack>
      <Button variant="outlined" sx={{ mt: drafts.length > 0 ? 2 : 0 }} disabled={drafts.length >= MAX_CONDITIONAL_ORDERS} onClick={onAdd}>
        {t('lineup.orders.add')}
      </Button>
    </Paper>
  )
}
