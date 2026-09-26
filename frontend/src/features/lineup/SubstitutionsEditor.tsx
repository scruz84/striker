import Button from '@mui/material/Button'
import FormControl from '@mui/material/FormControl'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Select from '@mui/material/Select'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { MAX_SCHEDULED_SUBSTITUTIONS, type Player } from '../../api/types'
import type { SubstitutionDraft } from './useSubstitutionDrafts'

export interface SubstitutionsEditorProps {
  starters: Player[]
  benchPlayers: Player[]
  unavailability: Map<number, unknown>
  drafts: SubstitutionDraft[]
  usedSubOutIds: Set<number | ''>
  usedSubInIds: Set<number | ''>
  onAdd: () => void
  onRemove: (index: number) => void
  onChange: (index: number, field: keyof SubstitutionDraft, value: number | '') => void
}

// The planned-substitutions section: pick who comes off, who comes on, and at what minute, up to
// MAX_SCHEDULED_SUBSTITUTIONS times. Shared by LineupEditorPage (the team's standing default
// lineup) and MatchLineupForm (a friendly's own per-match substitutions, see FriendlyMatchPage).
export function SubstitutionsEditor({
  starters,
  benchPlayers,
  unavailability,
  drafts,
  usedSubOutIds,
  usedSubInIds,
  onAdd,
  onRemove,
  onChange,
}: SubstitutionsEditorProps) {
  const { t } = useTranslation()

  return (
    <Paper variant="outlined" sx={{ p: { xs: 1.5, sm: 2 }, mt: { xs: 2, sm: 3 } }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('lineup.substitutions.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {t('lineup.substitutions.description', { max: MAX_SCHEDULED_SUBSTITUTIONS })}
      </Typography>
      <Stack spacing={2}>
        {drafts.map((draft, index) => (
          <Stack key={index} direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
            <FormControl size="small" sx={{ minWidth: { xs: '100%', sm: 180 } }}>
              <InputLabel id={`sub-out-label-${index}`}>{t('lineup.substitutions.playerOut')}</InputLabel>
              <Select
                labelId={`sub-out-label-${index}`}
                label={t('lineup.substitutions.playerOut')}
                value={draft.playerOutId}
                onChange={(e) => onChange(index, 'playerOutId', e.target.value as number)}
              >
                {starters
                  .filter((p) => p.id === draft.playerOutId || !usedSubOutIds.has(p.id))
                  .map((p) => (
                    <MenuItem key={p.id} value={p.id}>
                      {p.firstName} {p.lastName}
                    </MenuItem>
                  ))}
              </Select>
            </FormControl>
            <FormControl size="small" sx={{ minWidth: { xs: '100%', sm: 180 } }}>
              <InputLabel id={`sub-in-label-${index}`}>{t('lineup.substitutions.playerIn')}</InputLabel>
              <Select
                labelId={`sub-in-label-${index}`}
                label={t('lineup.substitutions.playerIn')}
                value={draft.playerInId}
                onChange={(e) => onChange(index, 'playerInId', e.target.value as number)}
              >
                {benchPlayers
                  .filter((p) => !unavailability.has(p.id))
                  .filter((p) => p.id === draft.playerInId || !usedSubInIds.has(p.id))
                  .map((p) => (
                    <MenuItem key={p.id} value={p.id}>
                      {p.firstName} {p.lastName}
                    </MenuItem>
                  ))}
              </Select>
            </FormControl>
            <TextField
              size="small"
              type="number"
              label={t('lineup.substitutions.minute')}
              value={draft.minute}
              onChange={(e) => {
                const value = e.target.value === '' ? '' : Number(e.target.value)
                onChange(index, 'minute', value)
              }}
              slotProps={{ htmlInput: { min: 1, max: 90 } }}
              sx={{ width: 100 }}
            />
            <Button size="small" color="secondary" onClick={() => onRemove(index)}>
              {t('lineup.substitutions.remove')}
            </Button>
          </Stack>
        ))}
      </Stack>
      <Button
        size="small"
        variant="outlined"
        sx={{ mt: drafts.length > 0 ? 2 : 0 }}
        disabled={drafts.length >= MAX_SCHEDULED_SUBSTITUTIONS}
        onClick={onAdd}
      >
        {t('lineup.substitutions.add')}
      </Button>
    </Paper>
  )
}
