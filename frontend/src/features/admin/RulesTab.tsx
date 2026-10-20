import { useState, type FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import ToggleButton from '@mui/material/ToggleButton'
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import {
  useAdminOverview,
  useAdminSetFreeAgentInjectionsPerWeek,
  useAdminSetFriendlySchedule,
  useAdminSetMaxFriendliesPerWeek,
  useAdminSetOfferNegotiationWindow,
  useAdminSetPromotionRelegation,
  useSetMatchSchedule,
} from '../../api/hooks'
import {
  WEEKDAY_OPTIONS,
} from '../../api/types'
import { localHourMinuteToSchedule, scheduleHourMinuteToLocal } from '../../utils/scheduleTime'

export function RulesTab() {
  const { data: overview, isLoading } = useAdminOverview(true)

  if (isLoading || !overview) {
    return <Skeleton variant="rectangular" height={100} sx={{ mb: 2 }} />
  }

  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Stack spacing={2}>
        <MatchScheduleForm overview={overview} />

        <PromotionRelegationForm overview={overview} />

        <MaxFriendliesPerWeekForm overview={overview} />

        <FriendlyScheduleForm overview={overview} />

        <FreeAgentInjectionsForm overview={overview} />

        <OfferNegotiationWindowForm overview={overview} />
      </Stack>
    </Paper>
  )
}

function MatchScheduleForm({ overview }: { overview: NonNullable<ReturnType<typeof useAdminOverview>['data']> }) {
  const { t } = useTranslation()
  const setMatchSchedule = useSetMatchSchedule()
  const [weekdaysMask, setWeekdaysMask] = useState(overview.matchScheduleWeekdaysMask)
  const localTime = scheduleHourMinuteToLocal(overview.matchScheduleHour, overview.matchScheduleMinute)
  const [time, setTime] = useState(`${String(localTime.hour).padStart(2, '0')}:${String(localTime.minute).padStart(2, '0')}`)

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    if (weekdaysMask === 0 || !time) {
      return
    }
    const [hourStr, minuteStr] = time.split(':')
    const { hour, minute } = localHourMinuteToSchedule(Number(hourStr), Number(minuteStr))
    setMatchSchedule.mutate({ weekdaysMask, hour, minute })
  }

  return (
    <Box component="form" onSubmit={handleSubmit}>
      <Typography variant="subtitle1" gutterBottom>
        {t('admin.matchSchedule.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
        {t('admin.matchSchedule.description')}
      </Typography>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
        <ToggleButtonGroup
          size="small"
          value={WEEKDAY_OPTIONS.filter((d) => (weekdaysMask & d.bit) !== 0).map((d) => d.bit)}
          onChange={(_event, newBits: number[]) => setWeekdaysMask(newBits.reduce((mask, bit) => mask | bit, 0))}
        >
          {WEEKDAY_OPTIONS.map((d) => (
            <ToggleButton key={d.bit} value={d.bit}>
              {t(`common.weekdaysShort.${d.key}`)}
            </ToggleButton>
          ))}
        </ToggleButtonGroup>
        <TextField
          label={t('admin.matchSchedule.time')}
          type="time"
          size="small"
          value={time}
          onChange={(e) => setTime(e.target.value)}
          sx={{ width: 140 }}
          slotProps={{ inputLabel: { shrink: true } }}
        />
        <Button type="submit" variant="contained" disabled={weekdaysMask === 0 || setMatchSchedule.isPending}>
          {t('admin.matchSchedule.save')}
        </Button>
      </Stack>
      {weekdaysMask === 0 && (
        <Typography variant="caption" color="error">
          {t('admin.matchSchedule.noWeekdaysError')}
        </Typography>
      )}
      {setMatchSchedule.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.matchSchedule.success')}</Alert>}
      {setMatchSchedule.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.matchSchedule.error')}</Alert>}
    </Box>
  )
}

function PromotionRelegationForm({ overview }: { overview: NonNullable<ReturnType<typeof useAdminOverview>['data']> }) {
  const { t } = useTranslation()
  const setPromotionRelegation = useAdminSetPromotionRelegation()
  const [promotionSlots, setPromotionSlots] = useState(String(overview.promotionSlotsPerGroup))
  const [relegationSlots, setRelegationSlots] = useState(String(overview.relegationSlotsPerGroup))

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    const promotionSlotsPerGroup = Number(promotionSlots)
    const relegationSlotsPerGroup = Number(relegationSlots)
    if (Number.isNaN(promotionSlotsPerGroup) || promotionSlotsPerGroup < 0) return
    if (Number.isNaN(relegationSlotsPerGroup) || relegationSlotsPerGroup < 0) return
    setPromotionRelegation.mutate({ promotionSlotsPerGroup, relegationSlotsPerGroup })
  }

  return (
    <Box component="form" onSubmit={handleSubmit}>
      <Typography variant="subtitle1" gutterBottom>
        {t('admin.promotionRelegation.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
        {t('admin.promotionRelegation.description')}
      </Typography>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
        <TextField
          label={t('admin.promotionRelegation.promoted')}
          type="number"
          size="small"
          value={promotionSlots}
          onChange={(e) => setPromotionSlots(e.target.value)}
          sx={{ width: 160 }}
          slotProps={{ htmlInput: { min: 0, max: 10 } }}
        />
        <TextField
          label={t('admin.promotionRelegation.relegated')}
          type="number"
          size="small"
          value={relegationSlots}
          onChange={(e) => setRelegationSlots(e.target.value)}
          sx={{ width: 160 }}
          slotProps={{ htmlInput: { min: 0, max: 10 } }}
        />
        <Button type="submit" variant="contained" disabled={setPromotionRelegation.isPending}>
          {t('admin.promotionRelegation.save')}
        </Button>
      </Stack>
      {setPromotionRelegation.isSuccess && (
        <Alert severity="success" sx={{ mt: 1 }}>
          {t('admin.promotionRelegation.success')}
        </Alert>
      )}
      {setPromotionRelegation.isError && (
        <Alert severity="error" sx={{ mt: 1 }}>
          {t('admin.promotionRelegation.error')}
        </Alert>
      )}
    </Box>
  )
}

function MaxFriendliesPerWeekForm({ overview }: { overview: NonNullable<ReturnType<typeof useAdminOverview>['data']> }) {
  const { t } = useTranslation()
  const setMaxFriendliesPerWeek = useAdminSetMaxFriendliesPerWeek()
  const [maxFriendliesPerWeek, setMaxFriendliesPerWeekValue] = useState(String(overview.maxFriendliesPerWeek))

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    const value = Number(maxFriendliesPerWeek)
    if (Number.isNaN(value) || value < 0) return
    setMaxFriendliesPerWeek.mutate(value)
  }

  return (
    <Box component="form" onSubmit={handleSubmit}>
      <Typography variant="subtitle1" gutterBottom>
        {t('admin.maxFriendlies.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
        {t('admin.maxFriendlies.description')}
      </Typography>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
        <TextField
          label={t('admin.maxFriendlies.perWeek')}
          type="number"
          size="small"
          value={maxFriendliesPerWeek}
          onChange={(e) => setMaxFriendliesPerWeekValue(e.target.value)}
          sx={{ width: 140 }}
          slotProps={{ htmlInput: { min: 0 } }}
        />
        <Button type="submit" variant="contained" disabled={setMaxFriendliesPerWeek.isPending}>
          {t('admin.maxFriendlies.save')}
        </Button>
      </Stack>
      {setMaxFriendliesPerWeek.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.maxFriendlies.success')}</Alert>}
      {setMaxFriendliesPerWeek.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.maxFriendlies.error')}</Alert>}
    </Box>
  )
}

function FriendlyScheduleForm({ overview }: { overview: NonNullable<ReturnType<typeof useAdminOverview>['data']> }) {
  const { t } = useTranslation()
  const setFriendlySchedule = useAdminSetFriendlySchedule()
  const [weekdaysMask, setWeekdaysMask] = useState(overview.friendlyScheduleWeekdays)

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    if (weekdaysMask === 0) return
    setFriendlySchedule.mutate(weekdaysMask)
  }

  return (
    <Box component="form" onSubmit={handleSubmit}>
      <Typography variant="subtitle1" gutterBottom>
        {t('admin.friendlySchedule.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
        {t('admin.friendlySchedule.description')}
      </Typography>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
        <ToggleButtonGroup
          size="small"
          value={WEEKDAY_OPTIONS.filter((d) => (weekdaysMask & d.bit) !== 0).map((d) => d.bit)}
          onChange={(_event, newBits: number[]) => setWeekdaysMask(newBits.reduce((mask, bit) => mask | bit, 0))}
        >
          {WEEKDAY_OPTIONS.map((d) => (
            <ToggleButton key={d.bit} value={d.bit}>
              {t(`common.weekdaysShort.${d.key}`)}
            </ToggleButton>
          ))}
        </ToggleButtonGroup>
        <Button type="submit" variant="contained" disabled={weekdaysMask === 0 || setFriendlySchedule.isPending}>
          {t('admin.friendlySchedule.save')}
        </Button>
      </Stack>
      {weekdaysMask === 0 && (
        <Typography variant="caption" color="error">
          {t('admin.friendlySchedule.noWeekdaysError')}
        </Typography>
      )}
      {setFriendlySchedule.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.friendlySchedule.success')}</Alert>}
      {setFriendlySchedule.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.friendlySchedule.error')}</Alert>}
    </Box>
  )
}

function FreeAgentInjectionsForm({ overview }: { overview: NonNullable<ReturnType<typeof useAdminOverview>['data']> }) {
  const { t } = useTranslation()
  const setFreeAgentInjectionsPerWeek = useAdminSetFreeAgentInjectionsPerWeek()
  const [freeAgentInjectionsPerWeek, setFreeAgentInjectionsPerWeekValue] = useState(String(overview.freeAgentInjectionsPerWeek))

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    const value = Number(freeAgentInjectionsPerWeek)
    if (Number.isNaN(value) || value < 0) return
    setFreeAgentInjectionsPerWeek.mutate(value)
  }

  return (
    <Box component="form" onSubmit={handleSubmit}>
      <Typography variant="subtitle1" gutterBottom>
        {t('admin.freeAgentInjections.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
        {t('admin.freeAgentInjections.description')}
      </Typography>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
        <TextField
          label={t('admin.freeAgentInjections.perWeek')}
          type="number"
          size="small"
          value={freeAgentInjectionsPerWeek}
          onChange={(e) => setFreeAgentInjectionsPerWeekValue(e.target.value)}
          sx={{ width: 140 }}
          slotProps={{ htmlInput: { min: 0 } }}
        />
        <Button type="submit" variant="contained" disabled={setFreeAgentInjectionsPerWeek.isPending}>
          {t('admin.freeAgentInjections.save')}
        </Button>
      </Stack>
      {setFreeAgentInjectionsPerWeek.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.freeAgentInjections.success')}</Alert>}
      {setFreeAgentInjectionsPerWeek.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.freeAgentInjections.error')}</Alert>}
    </Box>
  )
}

function OfferNegotiationWindowForm({ overview }: { overview: NonNullable<ReturnType<typeof useAdminOverview>['data']> }) {
  const { t } = useTranslation()
  const setOfferNegotiationWindow = useAdminSetOfferNegotiationWindow()
  const [offerNegotiationWindowMinutes, setOfferNegotiationWindowMinutesValue] = useState(String(overview.offerNegotiationWindowMinutes))

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    const value = Number(offerNegotiationWindowMinutes)
    if (Number.isNaN(value) || value < 0) return
    setOfferNegotiationWindow.mutate(value)
  }

  return (
    <Box component="form" onSubmit={handleSubmit}>
      <Typography variant="subtitle1" gutterBottom>
        {t('admin.offerNegotiationWindow.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
        {t('admin.offerNegotiationWindow.description')}
      </Typography>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
        <TextField
          label={t('admin.offerNegotiationWindow.minutes')}
          type="number"
          size="small"
          value={offerNegotiationWindowMinutes}
          onChange={(e) => setOfferNegotiationWindowMinutesValue(e.target.value)}
          sx={{ width: 140 }}
          slotProps={{ htmlInput: { min: 0 } }}
        />
        <Button type="submit" variant="contained" disabled={setOfferNegotiationWindow.isPending}>
          {t('admin.offerNegotiationWindow.save')}
        </Button>
      </Stack>
      {setOfferNegotiationWindow.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.offerNegotiationWindow.success')}</Alert>}
      {setOfferNegotiationWindow.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.offerNegotiationWindow.error')}</Alert>}
    </Box>
  )
}
