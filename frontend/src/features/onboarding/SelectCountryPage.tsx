import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Container from '@mui/material/Container'
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
import { useOnboardingCountries, useSelectCountry } from '../../api/hooks'

/** The mandatory first step for a brand-new sign-in (see LoggedUser.pendingCountrySelection) -
 * they pick which of the game's active countries to play in before anything else, and only then
 * gets a team assigned. Layout redirects here for as long as this is true, so there's nowhere
 * else for a pending user to end up. */
export function SelectCountryPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data: countries, isLoading } = useOnboardingCountries()
  const selectCountry = useSelectCountry()
  const [selected, setSelected] = useState('')

  const handleConfirm = () => {
    if (!selected) return
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
                    {country.name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
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
              disabled={!selected || selectCountry.isPending}
              onClick={handleConfirm}
            >
              {t('onboarding.confirm')}
            </Button>
          </Box>
        </Stack>
      </Paper>
    </Container>
  )
}
