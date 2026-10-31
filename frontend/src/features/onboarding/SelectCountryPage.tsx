import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Alert from '@mui/material/Alert'
import Checkbox from '@mui/material/Checkbox'
import Container from '@mui/material/Container'
import FormControlLabel from '@mui/material/FormControlLabel'
import FormControl from '@mui/material/FormControl'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Select, { type SelectChangeEvent } from '@mui/material/Select'
import Skeleton from '@mui/material/Skeleton'
import SportsSoccerIcon from '@mui/icons-material/SportsSoccer'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useAcceptTerms, useMe, useOnboardingCountries, useSelectCountry } from '../../api/hooks'
import { InactivityReleaseDays } from '../account/inactivityRules'
import { MINIMUM_AGE } from '../legal/legalMeta'
import { useSortedCountries } from '../../utils/countryOrder'

/** The mandatory first step for a brand-new sign-in (see LoggedUser.pendingCountrySelection) -
 * they pick which of the game's active countries to play in before anything else, and only then
 * gets a team assigned. Layout redirects here for as long as this is true, so there's nowhere
 * else for a pending user to end up. */
export function SelectCountryPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data, isLoading } = useOnboardingCountries()
  const { sorted: countries, countryName } = useSortedCountries(data, (country) => country.code)
  const selectCountry = useSelectCountry()
  const acceptTerms = useAcceptTerms()
  const { data: loggedUser } = useMe()
  const [selected, setSelected] = useState('')
  const [ageAndTerms, setAgeAndTerms] = useState(false)

  // Layout normally sends a user who has not accepted the terms to /accept-terms first; this is
  // the backstop (and the place the age/terms box lives for a first-time user who gets here
  // anyway): no club is assigned until it is ticked.
  const needsTerms = !!loggedUser && !loggedUser.termsAccepted
  const canConfirm = !!selected && (!needsTerms || ageAndTerms)

  const handleConfirm = async () => {
    if (!canConfirm) return
    if (needsTerms && loggedUser) {
      try {
        await acceptTerms.mutateAsync({ version: loggedUser.termsVersion, ageConfirmed: true })
      } catch {
        return
      }
    }
    selectCountry.mutate(selected, { onSuccess: () => navigate('/dashboard') })
  }

  return (
    <Container maxWidth="xs" sx={{ py: { xs: 6, sm: 10 } }}>
      <Paper variant="outlined" sx={{ p: 4 }}>
        <Stack spacing={2} sx={{ alignItems: 'center', textAlign: 'center' }}>
          <SportsSoccerIcon color="primary" sx={{ fontSize: 40 }} />
          <Typography variant="h6" sx={{ fontWeight: 600 }}>
            {t('onboarding.title')}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {t('onboarding.subtitle')}
          </Typography>

          {loggedUser?.releasedTeamName && (
            <Alert severity="info" sx={{ width: '100%', textAlign: 'left' }}>
              {t('onboarding.releasedNotice', { team: loggedUser.releasedTeamName, days: InactivityReleaseDays })}
            </Alert>
          )}

          {isLoading ? (
            <Skeleton variant="rectangular" width="100%" height={56} />
          ) : !countries?.length ? (
            <Typography variant="body2" color="text.secondary">
              {t('onboarding.noCountries')}
            </Typography>
          ) : (
            // A combo box rather than a radio group (issue #63) - a list of every country the
            // game can generate a world for is only going to grow, and a radio button per option
            // doesn't scale the way a searchable dropdown does.
            <FormControl fullWidth>
              <InputLabel id="onboarding-country-label">{t('onboarding.countryLabel')}</InputLabel>
              <Select
                labelId="onboarding-country-label"
                label={t('onboarding.countryLabel')}
                value={selected}
                onChange={(e: SelectChangeEvent) => setSelected(e.target.value)}
              >
                {countries.map((country) => (
                  <MenuItem key={country.code} value={country.code}>
                    {countryName(country.code)}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          )}

          {needsTerms && (
            <Box sx={{ width: '100%', textAlign: 'left' }}>
              <FormControlLabel
                control={<Checkbox checked={ageAndTerms} onChange={(e) => setAgeAndTerms(e.target.checked)} />}
                label={t('onboarding.ageAndTerms', { age: MINIMUM_AGE })}
              />
              {!ageAndTerms && !!selected && (
                <Typography variant="caption" color="text.secondary" component="div">
                  {t('onboarding.ageAndTermsRequired')}
                </Typography>
              )}
            </Box>
          )}

          {acceptTerms.isError && (
            <Typography variant="body2" color="error">
              {t('onboarding.termsError')}
            </Typography>
          )}

          {selectCountry.isError && (
            <Typography variant="body2" color="error">
              {t('onboarding.selectError')}
            </Typography>
          )}

          <Box sx={{ width: '100%' }}>
            <Button
              variant="contained"
              fullWidth
              disabled={!canConfirm || selectCountry.isPending || acceptTerms.isPending}
              onClick={() => void handleConfirm()}
            >
              {t('onboarding.confirm')}
            </Button>
          </Box>
        </Stack>
      </Paper>
    </Container>
  )
}
