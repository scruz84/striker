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
  useSetMatchSchedule,
} from '../../api/hooks'
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
      {[MatchScheduleForm, MaxFriendliesPerWeekForm, FriendlyScheduleForm, FreeAgentInjectionsForm, OfferNegotiationWindowForm].map(
        (Form, index) => (
          <Grid key={index} size={{ xs: 12, md: 6 }}>
            <Form overview={overview} />
          </Grid>
        ),
      )}
    </Grid>
  )
}

// MatchScheduleForm sets the days league matches are played. The kickoff time is not set here:
// each country's tier 1 kicks off at its own local league kickoff and every lower tier 15 minutes
// later (see the countries section of the World tab), so only the weekdays are saved; the stored
// hour and minute are kept as they are.
function MatchScheduleForm({ overview }: { overview: Overview }) {
  const { t } = useTranslation()
  const feedback = useAdminFeedback()
  const setMatchSchedule = useSetMatchSchedule()
  const [weekdaysMask, setWeekdaysMask] = useState(overview.matchScheduleWeekdaysMask)

  const handleSubmit = () => {
    setMatchSchedule.mutate(
      { weekdaysMask, hour: overview.matchScheduleHour, minute: overview.matchScheduleMinute },
      feedback('admin.matchSchedule.success', 'admin.matchSchedule.error'),
    )
  }

  return (
    <SettingCard
      title={t('admin.matchSchedule.title')}
      description={t('admin.matchSchedule.description')}
      submitLabel={t('admin.matchSchedule.save')}
      onSubmit={handleSubmit}
      pending={setMatchSchedule.isPending}
      invalid={weekdaysMask === 0}
      errorText={weekdaysMask === 0 ? t('admin.matchSchedule.noWeekdaysError') : undefined}
    >
      <WeekdayPicker value={weekdaysMask} onChange={setWeekdaysMask} />
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
