import { useEffect, useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Dialog from '@mui/material/Dialog'
import DialogActions from '@mui/material/DialogActions'
import DialogContent from '@mui/material/DialogContent'
import DialogTitle from '@mui/material/DialogTitle'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import ToggleButton from '@mui/material/ToggleButton'
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useChallengeToFriendly, useFriendlySchedule } from '../api/hooks'
import { defaultKickoffValue, isWeekdayAllowed, MIN_KICKOFF_LEAD_MINUTES } from '../utils/friendlyKickoff'

interface ChallengeToFriendlyDialogProps {
  opponentTeamId: number
  opponentTeamName: string
  open: boolean
  onClose: () => void
}

/** Lets the viewer challenge the team they're looking at (see TeamDetailsPage) to a friendly
 * match, choosing an optional proposed kickoff and which side of the fixture their own team
 * takes - the same request the Friendlies page's own challenge form sends, just pre-filled with
 * this team as the opponent. */
export function ChallengeToFriendlyDialog({ opponentTeamId, opponentTeamName, open, onClose }: ChallengeToFriendlyDialogProps) {
  const { t } = useTranslation()
  const { data: schedule } = useFriendlySchedule(open)
  const challenge = useChallengeToFriendly()
  const [proposedKickoff, setProposedKickoff] = useState('')
  const [challengerIsHome, setChallengerIsHome] = useState(true)

  useEffect(() => {
    if (open) {
      setProposedKickoff('')
      setChallengerIsHome(true)
      challenge.reset()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, opponentTeamId])

  const proposedKickoffAllowed = isWeekdayAllowed(schedule, proposedKickoff)

  const handleSubmit = () => {
    if (proposedKickoff && !proposedKickoffAllowed) return
    challenge.mutate({
      opponentTeamId,
      proposedKickoff: proposedKickoff ? new Date(proposedKickoff).toISOString() : undefined,
      challengerIsHome,
    })
  }

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{t('friendlies.challenge.dialogTitle', { name: opponentTeamName })}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          {challenge.isSuccess ? (
            <Alert severity="success">{t('friendlies.challenge.success')}</Alert>
          ) : (
            <>
              <Stack spacing={0.5}>
                <Typography variant="body2" color="text.secondary">
                  {t('friendlies.challenge.side')}
                </Typography>
                <ToggleButtonGroup
                  size="small"
                  exclusive
                  value={challengerIsHome}
                  onChange={(_event, value: boolean | null) => {
                    if (value !== null) setChallengerIsHome(value)
                  }}
                >
                  <ToggleButton value={true}>{t('friendlies.challenge.home')}</ToggleButton>
                  <ToggleButton value={false}>{t('friendlies.challenge.away')}</ToggleButton>
                </ToggleButtonGroup>
              </Stack>
              <TextField
                size="small"
                type="datetime-local"
                label={t('friendlies.challenge.proposedKickoff')}
                value={proposedKickoff}
                onChange={(e) => setProposedKickoff(e.target.value)}
                error={!proposedKickoffAllowed}
                helperText={!proposedKickoffAllowed ? t('friendlies.received.kickoffNotAllowed') : ' '}
                slotProps={{ inputLabel: { shrink: true }, htmlInput: { min: defaultKickoffValue(MIN_KICKOFF_LEAD_MINUTES) } }}
                fullWidth
              />
              {challenge.isError && <Alert severity="error">{t('friendlies.challenge.error')}</Alert>}
            </>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{challenge.isSuccess ? t('common.close') : t('common.cancel')}</Button>
        {!challenge.isSuccess && (
          <Button
            variant="contained"
            onClick={handleSubmit}
            disabled={challenge.isPending || (!!proposedKickoff && !proposedKickoffAllowed)}
          >
            {t('friendlies.challenge.button')}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  )
}
