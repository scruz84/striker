import { useState, type FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import FormControl from '@mui/material/FormControl'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Select from '@mui/material/Select'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import {
  useAdminAdjustBudget,
  useAdminInjectAllTeamsBudget,
  useAdminOverview,
  useAdminSetAiLevel,
} from '../../api/hooks'
import {
  AI_LEVEL_CARETAKER,
  AI_LEVEL_DIRECTOR,
  AI_LEVEL_MANAGER,
} from '../../api/types'
import { useConfirm } from '../../components/confirmDialogContext'
import { formatCurrency } from '../../utils/formatCurrency'

export function TeamsTab() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const { data: overview, isLoading } = useAdminOverview(true)
  const adjustBudget = useAdminAdjustBudget()
  const injectAllTeamsBudget = useAdminInjectAllTeamsBudget()
  const [teamId, setTeamId] = useState('')
  const [amount, setAmount] = useState('')
  const [description, setDescription] = useState('')
  const [allTeamsAmount, setAllTeamsAmount] = useState('')
  const [allTeamsDescription, setAllTeamsDescription] = useState('')

  const handleAdjustBudget = async (event: FormEvent) => {
    event.preventDefault()
    const teamIdNum = Number(teamId)
    const amountNum = Number(amount)
    if (!teamIdNum || !amountNum || !description) {
      return
    }
    const ok = await confirm({
      message: t('admin.adjustBudget.confirm', { teamId: teamIdNum, amount: formatCurrency(amountNum) }),
      confirmLabel: t('admin.adjustBudget.apply'),
      destructive: true,
    })
    if (!ok) {
      return
    }
    adjustBudget.mutate(
      { teamId: teamIdNum, amount: amountNum, description },
      { onSuccess: () => setDescription('') },
    )
  }

  const handleInjectAllTeamsBudget = async (event: FormEvent) => {
    event.preventDefault()
    const amountNum = Number(allTeamsAmount)
    if (!overview || !amountNum || !allTeamsDescription) {
      return
    }
    const ok = await confirm({
      message: t('admin.injectAllTeamsBudget.confirm', {
        amount: formatCurrency(amountNum),
        teamCount: overview.teamCount,
      }),
      confirmLabel: t('admin.injectAllTeamsBudget.apply'),
      destructive: true,
    })
    if (!ok) {
      return
    }
    injectAllTeamsBudget.mutate(
      { amount: amountNum, description: allTeamsDescription },
      { onSuccess: () => setAllTeamsDescription('') },
    )
  }

  if (isLoading || !overview) {
    return <Skeleton variant="rectangular" height={100} sx={{ mb: 2 }} />
  }

  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Stack spacing={2}>
        <Box component="form" onSubmit={handleAdjustBudget}>
          <Typography variant="subtitle1" gutterBottom>
            {t('admin.adjustBudget.title')}
          </Typography>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
            <TextField
              label={t('admin.adjustBudget.teamId')}
              size="small"
              value={teamId}
              onChange={(e) => setTeamId(e.target.value)}
              sx={{ width: 100 }}
            />
            <TextField
              label={t('admin.adjustBudget.amount')}
              size="small"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              helperText={t('admin.adjustBudget.amountHelper')}
              sx={{ width: 140 }}
            />
            <TextField
              label={t('admin.adjustBudget.description')}
              size="small"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              sx={{ width: 260 }}
            />
            <Button type="submit" variant="contained" disabled={adjustBudget.isPending}>
              {t('admin.adjustBudget.apply')}
            </Button>
          </Stack>
          {adjustBudget.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.adjustBudget.success')}</Alert>}
          {adjustBudget.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.adjustBudget.error')}</Alert>}
        </Box>

        <Box component="form" onSubmit={handleInjectAllTeamsBudget}>
          <Typography variant="subtitle1" gutterBottom>
            {t('admin.injectAllTeamsBudget.title')}
          </Typography>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
            <TextField
              label={t('admin.injectAllTeamsBudget.amount')}
              size="small"
              value={allTeamsAmount}
              onChange={(e) => setAllTeamsAmount(e.target.value)}
              helperText={t('admin.injectAllTeamsBudget.amountHelper')}
              sx={{ width: 140 }}
            />
            <TextField
              label={t('admin.injectAllTeamsBudget.description')}
              size="small"
              value={allTeamsDescription}
              onChange={(e) => setAllTeamsDescription(e.target.value)}
              sx={{ width: 260 }}
            />
            <Button type="submit" variant="contained" color="warning" disabled={injectAllTeamsBudget.isPending}>
              {t('admin.injectAllTeamsBudget.apply')}
            </Button>
          </Stack>
          {injectAllTeamsBudget.isSuccess && (
            <Alert severity="success" sx={{ mt: 1 }}>
              {t('admin.injectAllTeamsBudget.success', { teamCount: injectAllTeamsBudget.data.teamsAffected })}
            </Alert>
          )}
          {injectAllTeamsBudget.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.injectAllTeamsBudget.error')}</Alert>}
        </Box>

        <SetAiLevelForm />
      </Stack>
    </Paper>
  )
}

const AI_LEVEL_OPTIONS = [
  { value: AI_LEVEL_CARETAKER, labelKey: 'admin.setAiLevel.caretaker' },
  { value: AI_LEVEL_MANAGER, labelKey: 'admin.setAiLevel.manager' },
  { value: AI_LEVEL_DIRECTOR, labelKey: 'admin.setAiLevel.director' },
]

function SetAiLevelForm() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const setAiLevel = useAdminSetAiLevel()
  const [teamId, setTeamId] = useState('')
  const [aiLevel, setAiLevelValue] = useState<number>(AI_LEVEL_CARETAKER)

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    const teamIdNum = Number(teamId)
    if (!teamIdNum) {
      return
    }
    const ok = await confirm({
      message: t('admin.setAiLevel.confirm', { teamId: teamIdNum, level: aiLevel }),
      confirmLabel: t('admin.setAiLevel.apply'),
    })
    if (!ok) {
      return
    }
    setAiLevel.mutate({ teamId: teamIdNum, aiLevel })
  }

  return (
    <Box component="form" onSubmit={handleSubmit}>
      <Typography variant="subtitle1" gutterBottom>
        {t('admin.setAiLevel.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
        {t('admin.setAiLevel.description')}
      </Typography>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
        <TextField
          label={t('admin.setAiLevel.teamId')}
          size="small"
          value={teamId}
          onChange={(e) => setTeamId(e.target.value)}
          sx={{ width: 100 }}
        />
        <FormControl size="small" sx={{ width: 160 }}>
          <InputLabel id="ai-level-select-label">{t('admin.setAiLevel.level')}</InputLabel>
          <Select
            labelId="ai-level-select-label"
            label={t('admin.setAiLevel.level')}
            value={aiLevel}
            onChange={(e) => setAiLevelValue(Number(e.target.value))}
          >
            {AI_LEVEL_OPTIONS.map((option) => (
              <MenuItem key={option.value} value={option.value}>
                {t(option.labelKey)}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <Button type="submit" variant="contained" disabled={setAiLevel.isPending}>
          {t('admin.setAiLevel.apply')}
        </Button>
      </Stack>
      {setAiLevel.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.setAiLevel.success')}</Alert>}
      {setAiLevel.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.setAiLevel.error')}</Alert>}
    </Box>
  )
}
