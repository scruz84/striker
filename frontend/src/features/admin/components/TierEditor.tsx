import type { ReactNode } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { TierConfig } from '../../../api/types'
import { useIsPhone } from '../useIsPhone'
import { pyramidError, worldPlaces } from '../pyramidRules'

interface TierEditorProps {
  tiers: TierConfig[]
  onChange: (tiers: TierConfig[]) => void
  /** Buttons shown next to "Add tier", such as the save button. */
  actions?: ReactNode
}

// TierEditor edits a league structure (the global one or a country's own): one row per tier with
// its name, groups and teams per group, the pyramid-rule check and the places it holds. The tier
// rows stack as cards on a phone.
export function TierEditor({ tiers, onChange, actions }: TierEditorProps) {
  const { t } = useTranslation()
  const isPhone = useIsPhone()

  const updateTier = (index: number, patch: Partial<TierConfig>) =>
    onChange(tiers.map((tier, i) => (i === index ? { ...tier, ...patch } : tier)))

  const addTier = () => {
    // A new tier follows the pyramid rule: twice the groups of the deepest one, same group size.
    const deepest = tiers.reduce<TierConfig | undefined>((acc, tier) => (!acc || tier.tier > acc.tier ? tier : acc), undefined)
    const nextTierNumber = deepest ? deepest.tier + 1 : 1
    onChange([
      ...tiers,
      { tier: nextTierNumber, name: String(nextTierNumber), groups: deepest ? deepest.groups * 2 : 1, teamsPerLeague: deepest?.teamsPerLeague ?? 10 },
    ])
  }

  const removeTier = (index: number) => onChange(tiers.filter((_, i) => i !== index))
  const error = pyramidError(tiers)

  return (
    <>
      {isPhone ? (
        <Stack spacing={1.5}>
          {tiers.map((tier, index) => (
            <Paper key={index} variant="outlined" sx={{ p: 1.5 }}>
              <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
                <Typography variant="subtitle2">
                  {t('admin.worldStructure.tier')} {tier.tier}
                </Typography>
                <Button size="small" color="secondary" onClick={() => removeTier(index)}>
                  {t('admin.worldStructure.removeTier')}
                </Button>
              </Stack>
              <Stack spacing={1.5}>
                <TextField
                  label={t('admin.worldStructure.tierName')}
                  size="small"
                  fullWidth
                  value={tier.name}
                  onChange={(e) => updateTier(index, { name: e.target.value })}
                />
                <TextField
                  label={t('admin.worldStructure.groups')}
                  size="small"
                  type="number"
                  fullWidth
                  value={tier.groups}
                  onChange={(e) => updateTier(index, { groups: Number(e.target.value) })}
                />
                <TextField
                  label={t('admin.worldStructure.teamsPerLeague')}
                  size="small"
                  type="number"
                  fullWidth
                  value={tier.teamsPerLeague}
                  onChange={(e) => updateTier(index, { teamsPerLeague: Number(e.target.value) })}
                />
              </Stack>
            </Paper>
          ))}
        </Stack>
      ) : (
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>{t('admin.worldStructure.tier')}</TableCell>
                <TableCell>{t('admin.worldStructure.tierName')}</TableCell>
                <TableCell align="right">{t('admin.worldStructure.groups')}</TableCell>
                <TableCell align="right">{t('admin.worldStructure.teamsPerLeague')}</TableCell>
                <TableCell align="right" />
              </TableRow>
            </TableHead>
            <TableBody>
              {tiers.map((tier, index) => (
                <TableRow key={index}>
                  <TableCell>{tier.tier}</TableCell>
                  <TableCell>
                    <TextField
                      size="small"
                      value={tier.name}
                      onChange={(e) => updateTier(index, { name: e.target.value })}
                      sx={{ width: 80 }}
                    />
                  </TableCell>
                  <TableCell align="right">
                    <TextField
                      size="small"
                      type="number"
                      value={tier.groups}
                      onChange={(e) => updateTier(index, { groups: Number(e.target.value) })}
                      sx={{ width: 90 }}
                    />
                  </TableCell>
                  <TableCell align="right">
                    <TextField
                      size="small"
                      type="number"
                      value={tier.teamsPerLeague}
                      onChange={(e) => updateTier(index, { teamsPerLeague: Number(e.target.value) })}
                      sx={{ width: 90 }}
                    />
                  </TableCell>
                  <TableCell align="right">
                    <Button size="small" color="secondary" onClick={() => removeTier(index)}>
                      {t('admin.worldStructure.removeTier')}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      {error ? (
        <Alert severity="warning" sx={{ mt: 2 }}>
          {t(error.key, error.params)}
        </Alert>
      ) : (
        <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
          {t('admin.worldStructure.places', { count: worldPlaces(tiers) })}
        </Typography>
      )}
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} sx={{ mt: 2 }}>
        <Button size="small" onClick={addTier}>
          {t('admin.worldStructure.addTier')}
        </Button>
        {actions}
      </Stack>
    </>
  )
}
