import { useState } from 'react'
import FormControlLabel from '@mui/material/FormControlLabel'
import Grid from '@mui/material/Grid'
import Skeleton from '@mui/material/Skeleton'
import Switch from '@mui/material/Switch'
import TextField from '@mui/material/TextField'
import { useTranslation } from 'react-i18next'
import {
  useAdminOverview,
  useAdminSetFreeAgentInjectionsPerWeek,
  useAdminSetFriendlySchedule,
  useAdminSetInstantArchive,
  useAdminSetInstantRetention,
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
      {[
        MatchScheduleForm,
        MaxFriendliesPerWeekForm,
        FriendlyScheduleForm,
        FreeAgentInjectionsForm,
        OfferNegotiationWindowForm,
        InstantArchiveForm,
        InstantRetentionForm,
      ].map((Form, index) => (
        <Grid key={index} size={{ xs: 12, md: 6 }}>
          <Form overview={overview} />
        </Grid>
      ))}
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

// InstantArchiveForm turns the instant archive on or off and sets how many finished seasons keep
// their minute-by-minute detail (openspec/specs/match-data-retention). Off by default; what it
// deletes is gone for good, which the card says.
function InstantArchiveForm({ overview }: { overview: Overview }) {
  const { t } = useTranslation()
  const feedback = useAdminFeedback()
  const setInstantArchive = useAdminSetInstantArchive()
  const [enabled, setEnabled] = useState(overview.instantArchiveEnabled)
  const [keepSeasons, setKeepSeasons] = useState(String(overview.instantArchiveKeepSeasons))
  const value = Number(keepSeasons)
  const invalid = keepSeasons === '' || !Number.isInteger(value) || value < 0 || value > 10

  return (
    <SettingCard
      title={t('admin.instantArchive.title')}
      description={t('admin.instantArchive.description')}
      submitLabel={t('admin.instantArchive.save')}
      onSubmit={() =>
        setInstantArchive.mutate({ enabled, keepSeasons: value }, feedback('admin.instantArchive.success', 'admin.instantArchive.error'))
      }
      pending={setInstantArchive.isPending}
      invalid={invalid}
      errorText={invalid ? t('admin.instantArchive.rangeError') : undefined}
      color={enabled ? 'warning' : 'primary'}
    >
      <FormControlLabel
        control={<Switch checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />}
        label={t('admin.instantArchive.enabled')}
      />
      <TextField
        label={t('admin.instantArchive.keepSeasons')}
        type="number"
        size="small"
        value={keepSeasons}
        onChange={(e) => setKeepSeasons(e.target.value)}
        sx={fieldWidth}
        slotProps={{ htmlInput: { min: 0, max: 10 } }}
      />
    </SettingCard>
  )
}

// InstantRetentionForm sets how many days a played league match, cup tie or friendly keeps its
// minute-by-minute detail (openspec/specs/match-data-retention). It applies whether the season
// archive is on or off; 0 keeps the detail for good.
function InstantRetentionForm({ overview }: { overview: Overview }) {
  const { t } = useTranslation()
  const feedback = useAdminFeedback()
  const setRetention = useAdminSetInstantRetention()
  const [keepDays, setKeepDays] = useState(String(overview.instantKeepDays))
  const value = Number(keepDays)
  const invalid = keepDays === '' || !Number.isInteger(value) || value < 0 || value > 365

  return (
    <SettingCard
      title={t('admin.instantRetention.title')}
      description={t('admin.instantRetention.description')}
      submitLabel={t('admin.instantRetention.save')}
      onSubmit={() =>
        setRetention.mutate(value, feedback('admin.instantRetention.success', 'admin.instantRetention.error'))
      }
      pending={setRetention.isPending}
      invalid={invalid}
      errorText={invalid ? t('admin.instantRetention.rangeError') : undefined}
    >
      <TextField
        label={t('admin.instantRetention.keepDays')}
        type="number"
        size="small"
        value={keepDays}
        onChange={(e) => setKeepDays(e.target.value)}
        sx={fieldWidth}
        slotProps={{ htmlInput: { min: 0, max: 365 } }}
      />
    </SettingCard>
  )
}
