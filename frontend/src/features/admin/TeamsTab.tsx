import { useState } from 'react'
import FormControl from '@mui/material/FormControl'
import Grid from '@mui/material/Grid'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Select from '@mui/material/Select'
import Skeleton from '@mui/material/Skeleton'
import TextField from '@mui/material/TextField'
import { useTranslation } from 'react-i18next'
import { useAdminAdjustBudget, useAdminInjectAllTeamsBudget, useAdminOverview, useAdminSetAiLevel } from '../../api/hooks'
import { AI_LEVEL_CARETAKER, AI_LEVEL_DIRECTOR, AI_LEVEL_MANAGER } from '../../api/types'
import { useConfirm } from '../../components/confirmDialogContext'
import { useNotify } from '../../components/notifyContext'
import { formatCurrency } from '../../utils/formatCurrency'
import { SettingCard } from './components/SettingCard'
import { TeamPicker, type PickedTeam } from './components/TeamPicker'
import { useAdminFeedback } from './useAdminFeedback'

const fieldWidth = { width: { xs: '100%', sm: 140 } }
const wideFieldWidth = { width: { xs: '100%', sm: 260 } }
const teamFieldWidth = { width: { xs: '100%', sm: 300 } }

export function TeamsTab() {
  const { data: overview, isLoading } = useAdminOverview(true)

  if (isLoading || !overview) {
    return <Skeleton variant="rectangular" height={100} sx={{ mb: 2 }} />
  }

  return (
    <Grid container spacing={2} sx={{ mb: 2 }}>
      <Grid size={{ xs: 12, md: 6 }}>
        <AdjustBudgetForm />
      </Grid>
      <Grid size={{ xs: 12, md: 6 }}>
        <InjectAllTeamsBudgetForm teamCount={overview.teamCount} />
      </Grid>
      <Grid size={{ xs: 12, md: 6 }}>
        <SetAiLevelForm />
      </Grid>
    </Grid>
  )
}

function AdjustBudgetForm() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const notify = useNotify()
  const adjustBudget = useAdminAdjustBudget()
  const [team, setTeam] = useState<PickedTeam | null>(null)
  const [amount, setAmount] = useState('')
  const [description, setDescription] = useState('')
  const amountNum = Number(amount)

  const handleSubmit = async () => {
    if (!team) {
      return
    }
    const ok = await confirm({
      message: t('admin.adjustBudget.confirm', { team: team?.label, amount: formatCurrency(amountNum) }),
      confirmLabel: t('admin.adjustBudget.apply'),
      destructive: true,
    })
    if (!ok) {
      return
    }
    adjustBudget.mutate(
      { teamId: team.id, amount: amountNum, description },
      {
        onSuccess: () => {
          notify.success(t('admin.adjustBudget.success'))
          setDescription('')
        },
        onError: () => notify.error(t('admin.adjustBudget.error')),
      },
    )
  }

  return (
    <SettingCard
      title={t('admin.adjustBudget.title')}
      submitLabel={t('admin.adjustBudget.apply')}
      onSubmit={handleSubmit}
      pending={adjustBudget.isPending}
      invalid={!team || !amountNum || !description}
    >
      <TeamPicker label={t('admin.adjustBudget.teamId')} value={team} onChange={setTeam} sx={teamFieldWidth} />
      <TextField
        label={t('admin.adjustBudget.amount')}
        size="small"
        value={amount}
        onChange={(e) => setAmount(e.target.value)}
        helperText={t('admin.adjustBudget.amountHelper')}
        sx={fieldWidth}
      />
      <TextField label={t('admin.adjustBudget.description')} size="small" value={description} onChange={(e) => setDescription(e.target.value)} sx={wideFieldWidth} />
    </SettingCard>
  )
}

function InjectAllTeamsBudgetForm({ teamCount }: { teamCount: number }) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const notify = useNotify()
  const injectAllTeamsBudget = useAdminInjectAllTeamsBudget()
  const [amount, setAmount] = useState('')
  const [description, setDescription] = useState('')
  const amountNum = Number(amount)

  const handleSubmit = async () => {
    const ok = await confirm({
      message: t('admin.injectAllTeamsBudget.confirm', { amount: formatCurrency(amountNum), teamCount }),
      confirmLabel: t('admin.injectAllTeamsBudget.apply'),
      destructive: true,
    })
    if (!ok) {
      return
    }
    injectAllTeamsBudget.mutate(
      { amount: amountNum, description },
      {
        onSuccess: (result) => {
          notify.success(t('admin.injectAllTeamsBudget.success', { teamCount: result.teamsAffected }))
          setDescription('')
        },
        onError: () => notify.error(t('admin.injectAllTeamsBudget.error')),
      },
    )
  }

  return (
    <SettingCard
      title={t('admin.injectAllTeamsBudget.title')}
      submitLabel={t('admin.injectAllTeamsBudget.apply')}
      color="warning"
      onSubmit={handleSubmit}
      pending={injectAllTeamsBudget.isPending}
      invalid={!amountNum || !description}
    >
      <TextField
        label={t('admin.injectAllTeamsBudget.amount')}
        size="small"
        value={amount}
        onChange={(e) => setAmount(e.target.value)}
        helperText={t('admin.injectAllTeamsBudget.amountHelper')}
        sx={fieldWidth}
      />
      <TextField
        label={t('admin.injectAllTeamsBudget.description')}
        size="small"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        sx={wideFieldWidth}
      />
    </SettingCard>
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
  const feedback = useAdminFeedback()
  const setAiLevel = useAdminSetAiLevel()
  const [team, setTeam] = useState<PickedTeam | null>(null)
  const [aiLevel, setAiLevelValue] = useState<number>(AI_LEVEL_CARETAKER)

  const handleSubmit = async () => {
    if (!team) {
      return
    }
    const ok = await confirm({
      message: t('admin.setAiLevel.confirm', { team: team.label, level: aiLevel }),
      confirmLabel: t('admin.setAiLevel.apply'),
    })
    if (!ok) {
      return
    }
    setAiLevel.mutate({ teamId: team.id, aiLevel }, feedback('admin.setAiLevel.success', 'admin.setAiLevel.error'))
  }

  return (
    <SettingCard
      title={t('admin.setAiLevel.title')}
      description={t('admin.setAiLevel.description')}
      submitLabel={t('admin.setAiLevel.apply')}
      onSubmit={handleSubmit}
      pending={setAiLevel.isPending}
      invalid={!team}
    >
      <TeamPicker label={t('admin.setAiLevel.teamId')} value={team} onChange={setTeam} sx={teamFieldWidth} />
      <FormControl size="small" sx={{ width: { xs: '100%', sm: 160 } }}>
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
    </SettingCard>
  )
}
