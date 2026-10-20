import { useState } from 'react'
import Grid from '@mui/material/Grid'
import Skeleton from '@mui/material/Skeleton'
import TextField from '@mui/material/TextField'
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
import { localHourMinuteToSchedule, scheduleHourMinuteToLocal } from '../../utils/scheduleTime'
import { SettingCard } from './components/SettingCard'
import { WeekdayPicker } from './components/WeekdayPicker'
import { useAdminFeedback } from './useAdminFeedback'

type Overview = NonNullable<ReturnType<typeof useAdminOverview>['data']>

const fieldWidth = { width: { xs: '100%', sm: 140 } }

// The global rules of the game world, one self-contained card each (two per row from `md`), every
// card saving on its own.
export function RulesTab() {
  const { data: overview, isLoading } = useAdminOverview(true)

  if (isLoading || !overview) {
    return <Skeleton variant="rectangular" height={100} sx={{ mb: 2 }} />
  }

  return (
    <Grid container spacing={2} sx={{ mb: 2 }}>
      {[MatchScheduleForm, PromotionRelegationForm, MaxFriendliesPerWeekForm, FriendlyScheduleForm, FreeAgentInjectionsForm, OfferNegotiationWindowForm].map(
        (Form, index) => (
          <Grid key={index} size={{ xs: 12, md: 6 }}>
            <Form overview={overview} />
          </Grid>
        ),
      )}
    </Grid>
  )
}

function formatTime(hour: number, minute: number) {
  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`
}

function MatchScheduleForm({ overview }: { overview: Overview }) {
  const { t } = useTranslation()
  const feedback = useAdminFeedback()
  const setMatchSchedule = useSetMatchSchedule()
  const [weekdaysMask, setWeekdaysMask] = useState(overview.matchScheduleWeekdaysMask)
  const localTime = scheduleHourMinuteToLocal(overview.matchScheduleHour, overview.matchScheduleMinute)
  const [time, setTime] = useState(formatTime(localTime.hour, localTime.minute))

  const handleSubmit = () => {
    const [hourStr, minuteStr] = time.split(':')
    const { hour, minute } = localHourMinuteToSchedule(Number(hourStr), Number(minuteStr))
    setMatchSchedule.mutate({ weekdaysMask, hour, minute }, feedback('admin.matchSchedule.success', 'admin.matchSchedule.error'))
  }

  return (
    <SettingCard
      title={t('admin.matchSchedule.title')}
      description={t('admin.matchSchedule.description')}
      submitLabel={t('admin.matchSchedule.save')}
      onSubmit={handleSubmit}
      pending={setMatchSchedule.isPending}
      invalid={weekdaysMask === 0 || !time}
      errorText={weekdaysMask === 0 ? t('admin.matchSchedule.noWeekdaysError') : undefined}
    >
      <WeekdayPicker value={weekdaysMask} onChange={setWeekdaysMask} />
      <TextField
        label={t('admin.matchSchedule.time')}
        type="time"
        size="small"
        value={time}
        onChange={(e) => setTime(e.target.value)}
        sx={fieldWidth}
        slotProps={{ inputLabel: { shrink: true } }}
      />
    </SettingCard>
  )
}

function PromotionRelegationForm({ overview }: { overview: Overview }) {
  const { t } = useTranslation()
  const feedback = useAdminFeedback()
  const setPromotionRelegation = useAdminSetPromotionRelegation()
  const [promotionSlots, setPromotionSlots] = useState(String(overview.promotionSlotsPerGroup))
  const [relegationSlots, setRelegationSlots] = useState(String(overview.relegationSlotsPerGroup))

  const promotionSlotsPerGroup = Number(promotionSlots)
  const relegationSlotsPerGroup = Number(relegationSlots)
  const invalid =
    Number.isNaN(promotionSlotsPerGroup) || promotionSlotsPerGroup < 0 || Number.isNaN(relegationSlotsPerGroup) || relegationSlotsPerGroup < 0

  return (
    <SettingCard
      title={t('admin.promotionRelegation.title')}
      description={t('admin.promotionRelegation.description')}
      submitLabel={t('admin.promotionRelegation.save')}
      onSubmit={() =>
        setPromotionRelegation.mutate(
          { promotionSlotsPerGroup, relegationSlotsPerGroup },
          feedback('admin.promotionRelegation.success', 'admin.promotionRelegation.error'),
        )
      }
      pending={setPromotionRelegation.isPending}
      invalid={invalid}
    >
      <TextField
        label={t('admin.promotionRelegation.promoted')}
        type="number"
        size="small"
        value={promotionSlots}
        onChange={(e) => setPromotionSlots(e.target.value)}
        sx={fieldWidth}
        slotProps={{ htmlInput: { min: 0, max: 10 } }}
      />
      <TextField
        label={t('admin.promotionRelegation.relegated')}
        type="number"
        size="small"
        value={relegationSlots}
        onChange={(e) => setRelegationSlots(e.target.value)}
        sx={fieldWidth}
        slotProps={{ htmlInput: { min: 0, max: 10 } }}
      />
    </SettingCard>
  )
}

function MaxFriendliesPerWeekForm({ overview }: { overview: Overview }) {
  const { t } = useTranslation()
  const feedback = useAdminFeedback()
  const setMaxFriendliesPerWeek = useAdminSetMaxFriendliesPerWeek()
  const [maxFriendliesPerWeek, setMaxFriendliesPerWeekValue] = useState(String(overview.maxFriendliesPerWeek))
  const value = Number(maxFriendliesPerWeek)

  return (
    <SettingCard
      title={t('admin.maxFriendlies.title')}
      description={t('admin.maxFriendlies.description')}
      submitLabel={t('admin.maxFriendlies.save')}
      onSubmit={() => setMaxFriendliesPerWeek.mutate(value, feedback('admin.maxFriendlies.success', 'admin.maxFriendlies.error'))}
      pending={setMaxFriendliesPerWeek.isPending}
      invalid={Number.isNaN(value) || value < 0}
    >
      <TextField
        label={t('admin.maxFriendlies.perWeek')}
        type="number"
        size="small"
        value={maxFriendliesPerWeek}
        onChange={(e) => setMaxFriendliesPerWeekValue(e.target.value)}
        sx={fieldWidth}
        slotProps={{ htmlInput: { min: 0 } }}
      />
    </SettingCard>
  )
}

function FriendlyScheduleForm({ overview }: { overview: Overview }) {
  const { t } = useTranslation()
  const feedback = useAdminFeedback()
  const setFriendlySchedule = useAdminSetFriendlySchedule()
  const [weekdaysMask, setWeekdaysMask] = useState(overview.friendlyScheduleWeekdays)

  return (
    <SettingCard
      title={t('admin.friendlySchedule.title')}
      description={t('admin.friendlySchedule.description')}
      submitLabel={t('admin.friendlySchedule.save')}
      onSubmit={() => setFriendlySchedule.mutate(weekdaysMask, feedback('admin.friendlySchedule.success', 'admin.friendlySchedule.error'))}
      pending={setFriendlySchedule.isPending}
      invalid={weekdaysMask === 0}
      errorText={weekdaysMask === 0 ? t('admin.friendlySchedule.noWeekdaysError') : undefined}
    >
      <WeekdayPicker value={weekdaysMask} onChange={setWeekdaysMask} />
    </SettingCard>
  )
}

function FreeAgentInjectionsForm({ overview }: { overview: Overview }) {
  const { t } = useTranslation()
  const feedback = useAdminFeedback()
  const setFreeAgentInjectionsPerWeek = useAdminSetFreeAgentInjectionsPerWeek()
  const [freeAgentInjectionsPerWeek, setFreeAgentInjectionsPerWeekValue] = useState(String(overview.freeAgentInjectionsPerWeek))
  const value = Number(freeAgentInjectionsPerWeek)

  return (
    <SettingCard
      title={t('admin.freeAgentInjections.title')}
      description={t('admin.freeAgentInjections.description')}
      submitLabel={t('admin.freeAgentInjections.save')}
      onSubmit={() =>
        setFreeAgentInjectionsPerWeek.mutate(value, feedback('admin.freeAgentInjections.success', 'admin.freeAgentInjections.error'))
      }
      pending={setFreeAgentInjectionsPerWeek.isPending}
      invalid={Number.isNaN(value) || value < 0}
    >
      <TextField
        label={t('admin.freeAgentInjections.perWeek')}
        type="number"
        size="small"
        value={freeAgentInjectionsPerWeek}
        onChange={(e) => setFreeAgentInjectionsPerWeekValue(e.target.value)}
        sx={fieldWidth}
        slotProps={{ htmlInput: { min: 0 } }}
      />
    </SettingCard>
  )
}

function OfferNegotiationWindowForm({ overview }: { overview: Overview }) {
  const { t } = useTranslation()
  const feedback = useAdminFeedback()
  const setOfferNegotiationWindow = useAdminSetOfferNegotiationWindow()
  const [offerNegotiationWindowMinutes, setOfferNegotiationWindowMinutesValue] = useState(String(overview.offerNegotiationWindowMinutes))
  const value = Number(offerNegotiationWindowMinutes)

  return (
    <SettingCard
      title={t('admin.offerNegotiationWindow.title')}
      description={t('admin.offerNegotiationWindow.description')}
      submitLabel={t('admin.offerNegotiationWindow.save')}
      onSubmit={() =>
        setOfferNegotiationWindow.mutate(value, feedback('admin.offerNegotiationWindow.success', 'admin.offerNegotiationWindow.error'))
      }
      pending={setOfferNegotiationWindow.isPending}
      invalid={Number.isNaN(value) || value < 0}
    >
      <TextField
        label={t('admin.offerNegotiationWindow.minutes')}
        type="number"
        size="small"
        value={offerNegotiationWindowMinutes}
        onChange={(e) => setOfferNegotiationWindowMinutesValue(e.target.value)}
        sx={fieldWidth}
        slotProps={{ htmlInput: { min: 0 } }}
      />
    </SettingCard>
  )
}
