import { useState, type FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardActionArea from '@mui/material/CardActionArea'
import CardContent from '@mui/material/CardContent'
import Chip from '@mui/material/Chip'
import EditIcon from '@mui/icons-material/Edit'
import Grid from '@mui/material/Grid'
import IconButton from '@mui/material/IconButton'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import { useFinances, useMe, useRenameStadium, useSetSponsorDeal, useSetTicketPrice, useUpgradeStadium } from '../../api/hooks'
import {
  calculateStadiumMaintenanceCost,
  calculateStadiumUpgradeCost,
  DEFAULT_STADIUM_UPGRADE_SEATS,
  MAX_STADIUM_UPGRADE_SEATS,
  MIN_STADIUM_UPGRADE_SEATS,
  MIN_TICKET_PRICE,
  SPONSOR_DEAL_BALANCED,
  SPONSOR_DEAL_DESCRIPTION_KEYS,
  SPONSOR_DEAL_LABEL_KEYS,
  SPONSOR_DEAL_PERFORMANCE,
  SPONSOR_DEAL_STEADY,
  type SponsorDealOption,
  type Stadium,
} from '../../api/types'
import { useConfirm } from '../../components/confirmDialogContext'
import { formatCurrency, formatNumber } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'
import { NumberField } from '../../components/NumberField'
import { ResponsiveDialog } from '../../components/ResponsiveDialog'
import { ratingTierColor } from '../../utils/playerRating'
import { BudgetOverTimeChart } from './BudgetOverTimeChart'
import { CashFlowKpis } from './CashFlowForecastCard'
import { IncomeBreakdownChart } from './IncomeBreakdownChart'
import { TransactionLedger } from './TransactionLedger'
import { PageHeader } from '../../components/PageHeader'

export function FinancesPage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: finances, isLoading } = useFinances(!!loggedUser)

  if (isLoading || !finances) {
    return <Skeleton variant="rectangular" height={300} />
  }

  // Phone order (top to bottom): KPIs -> stadium -> sponsorship -> income breakdown -> chart ->
  // movements. The weekly forecast lives in the KPI grid instead of a card of its own.
  return (
    <Box>
      <PageHeader title={t('finances.title')} />

      {finances.insolvent && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {t('finances.insolvent')}
        </Alert>
      )}

      <CashFlowKpis transactions={finances.transactions} budget={finances.budget} maxDebt={finances.maxDebt} />
      <Typography variant="caption" color="text.secondary" component="p" sx={{ mb: 2 }}>
        {t('finances.wageFactor', { factor: finances.wageFactor })}
      </Typography>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, sm: 8 }}>{finances.stadium && (
            <StadiumCard
              stadium={finances.stadium}
              recommendedTicketPrice={finances.recommendedTicketPrice}
              maxTicketPrice={finances.maxTicketPrice}
            />
          )}</Grid>
        <Grid size={{ xs: 12, sm: 4 }}>
          <FanbaseCard fanbase={finances.fanbase} socios={finances.socios} />
        </Grid>
      </Grid>

      <SponsorDealCard sponsorDeal={finances.sponsorDeal} options={finances.sponsorDealOptions} locked={finances.sponsorDealLocked} />

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 6 }}>
          <IncomeBreakdownChart transactions={finances.transactions} />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <BudgetOverTimeChart transactions={finances.transactions} budget={finances.budget} />
        </Grid>
      </Grid>

      <TransactionLedger transactions={finances.transactions} />
    </Box>
  )
}

// Lets the owner pick between a higher guaranteed per-match sponsorship payment (Steady) and a
// lower guaranteed payment made up for by a bigger win/draw bonus (Performance), with Balanced in
// between - see backend engine.sponsorshipAmount for the actual multipliers. Takes effect from
// either side's next match, same as ticket price.
function SponsorDealCard({
  sponsorDeal,
  options,
  locked,
}: {
  sponsorDeal: number
  options: SponsorDealOption[]
  locked: boolean
}) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const setSponsorDeal = useSetSponsorDeal()
  // Shows the just-clicked option as selected right away, while the confirm dialog below is
  // open - sponsorDeal is the server-confirmed value, so without this the highlight would just
  // sit on the old selection until (if) the choice is confirmed. Reset back to null either way
  // once the dialog settles, so the cards fall back to reflecting sponsorDeal.
  const [pendingDeal, setPendingDeal] = useState<number | null>(null)
  const amountsByDeal = new Map(options.map((option) => [option.deal, option]))
  const selectedDeal = pendingDeal ?? sponsorDeal

  const handleSelect = async (deal: number) => {
    if (deal === sponsorDeal) {
      return
    }
    setPendingDeal(deal)
    const ok = await confirm({
      message: t('finances.sponsorDeal.confirmChange'),
      confirmLabel: t('finances.sponsorDeal.confirmLabel'),
    })
    if (ok) {
      setSponsorDeal.mutate(deal, { onSettled: () => setPendingDeal(null) })
    } else {
      setPendingDeal(null)
    }
  }

  return (
    <Box sx={{ mb: 2 }}>
      <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600, mb: 0.5 }}>
        {t('finances.sponsorDeal.title')}
      </Typography>
      {locked && (
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
          {t('finances.sponsorDeal.locked')}
        </Typography>
      )}
      <Grid container spacing={1.5} role="radiogroup" aria-label={t('finances.sponsorDeal.title')}>
        {[SPONSOR_DEAL_STEADY, SPONSOR_DEAL_BALANCED, SPONSOR_DEAL_PERFORMANCE].map((deal) => {
          const amounts = amountsByDeal.get(deal)
          const selected = selectedDeal === deal
          return (
            <Grid key={deal} size={{ xs: 12, sm: 4 }}>
              <Card
                variant="outlined"
                sx={{ height: '100%', borderColor: selected ? 'primary.main' : undefined, borderWidth: selected ? 2 : 1, bgcolor: selected ? 'primaryContainer' : undefined }}
              >
                <CardActionArea
                  role="radio"
                  aria-checked={selected}
                  disabled={setSponsorDeal.isPending || locked}
                  onClick={() => void handleSelect(deal)}
                  sx={{ height: '100%', p: 1.5, display: 'flex', flexDirection: 'column', alignItems: 'stretch' }}
                >
                  <Stack direction="row" sx={{ justifyContent: 'space-between', alignItems: 'center' }}>
                    <Typography variant="subtitle2">{t(SPONSOR_DEAL_LABEL_KEYS[deal])}</Typography>
                    {selected && <Chip size="small" color="primary" label={t('finances.sponsorDeal.selected')} />}
                  </Stack>
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                    {t(SPONSOR_DEAL_DESCRIPTION_KEYS[deal])}
                  </Typography>
                  {amounts && (
                    <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 1, mt: 'auto' }}>
                      {(
                        [
                          ['win', amounts.win],
                          ['draw', amounts.draw],
                          ['loss', amounts.loss],
                        ] as const
                      ).map(([outcome, amount]) => (
                        <Box key={outcome} sx={{ textAlign: 'center' }} title={t(`finances.sponsorDeal.${outcome}Long`)}>
                          <Typography variant="chipLabel" color="text.secondary" component="div">
                            {t(`finances.sponsorDeal.${outcome}`)}
                          </Typography>
                          <Typography variant="body2" sx={{ fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>
                            {formatCurrency(amount, { compact: true })}
                          </Typography>
                        </Box>
                      ))}
                    </Box>
                  )}
                </CardActionArea>
              </Card>
            </Grid>
          )
        })}
      </Grid>
      {setSponsorDeal.isError && (
        <Alert severity="error" sx={{ mt: 1 }}>
          {t('finances.sponsorDeal.error')}
        </Alert>
      )}
    </Box>
  )
}

// FanbaseCard shows this team's current 0-99 supporter-base index (see backend common.GetFanbase)
// - read-only, unlike SponsorDealCard/StadiumCard, since Fanbase drifts on its own from match
// results rather than being something a manager directly sets.
function FanbaseCard({ fanbase, socios }: { fanbase: number; socios: number }) {
  const { t } = useTranslation()
  const theme = useTheme()

  return (
    <Card sx={{ height: '100%' }}>
      <CardContent>
        <Typography variant="overline" color="text.secondary">
          {t('finances.fanbase.title')}
        </Typography>
        <Typography variant="h4" sx={{ color: ratingTierColor(fanbase, theme.palette) }}>
          {fanbase}
        </Typography>
        <Typography variant="caption" color="text.secondary" component="p">
          {t('finances.fanbase.description')}
        </Typography>
        <Typography variant="body2" sx={{ mt: 1 }}>
          {t('finances.socios.count', { count: socios })}
        </Typography>
        <Typography variant="caption" color="text.secondary">
          {t('finances.socios.description')}
        </Typography>
      </CardContent>
    </Card>
  )
}

export function StadiumCard({
  stadium,
  recommendedTicketPrice,
  maxTicketPrice,
}: {
  stadium: Stadium
  recommendedTicketPrice: number
  maxTicketPrice: number
}) {
  const { t } = useTranslation()
  const setTicketPrice = useSetTicketPrice()
  const upgradeStadium = useUpgradeStadium()
  const renameStadium = useRenameStadium()
  const [dialog, setDialog] = useState<'expand' | 'price' | null>(null)
  const [price, setPrice] = useState(String(stadium.ticketPrice))
  const [seats, setSeats] = useState(String(DEFAULT_STADIUM_UPGRADE_SEATS))
  const [editingName, setEditingName] = useState(false)
  const [name, setName] = useState(stadium.name)

  const seatsNum = Number(seats)
  const seatsValid = Number.isInteger(seatsNum) && seatsNum >= MIN_STADIUM_UPGRADE_SEATS && seatsNum <= MAX_STADIUM_UPGRADE_SEATS
  const expandCost = seatsValid ? calculateStadiumUpgradeCost(stadium.capacity, seatsNum) : null
  const upgradePending = stadium.pendingCapacity != null && stadium.upgradeCompletesAt != null

  const priceNum = Number(price)
  const priceValid = Number.isInteger(priceNum) && priceNum >= MIN_TICKET_PRICE && priceNum <= maxTicketPrice

  const handlePriceSubmit = (event: FormEvent) => {
    event.preventDefault()
    if (!priceValid) {
      return
    }
    setTicketPrice.mutate(priceNum, { onSuccess: () => setDialog(null) })
  }

  const handleUpgrade = () => {
    if (!seatsValid || expandCost == null) {
      return
    }
    upgradeStadium.mutate(seatsNum, { onSuccess: () => setDialog(null) })
  }

  const handleRenameSubmit = (event: FormEvent) => {
    event.preventDefault()
    const trimmed = name.trim()
    if (!trimmed || trimmed === stadium.name) {
      setEditingName(false)
      return
    }
    renameStadium.mutate(trimmed, { onSuccess: () => setEditingName(false) })
  }

  return (
    <Card sx={{ height: '100%' }}>
      <CardContent>
        <Typography variant="overline" color="text.secondary">
          {t('finances.stadium')}
        </Typography>
        {editingName ? (
          <Box component="form" onSubmit={handleRenameSubmit} sx={{ mb: 0.5 }}>
            <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
              <TextField
                label={t('finances.stadiumNameLabel')}
                size="small"
                autoFocus
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
              <Button type="submit" size="small" variant="outlined" disabled={renameStadium.isPending}>
                {t('common.save')}
              </Button>
            </Stack>
          </Box>
        ) : (
          <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
            <Typography variant="subtitle1">{stadium.name}</Typography>
            <IconButton
              size="small"
              aria-label={t('finances.renameStadium')}
              onClick={() => {
                setName(stadium.name)
                setEditingName(true)
              }}
            >
              <EditIcon fontSize="inherit" />
            </IconButton>
          </Stack>
        )}
        {renameStadium.isError && (
          <Alert severity="error" sx={{ mb: 1 }}>
            {t('finances.renameStadiumError')}
          </Alert>
        )}
        <Stack direction="row" spacing={1} sx={{ alignItems: 'baseline' }}>
          <Typography variant="stat" sx={{ fontSize: '1.75rem' }}>
            {formatNumber(stadium.capacity)}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            {t('finances.seats')}
          </Typography>
        </Stack>
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1.5 }}>
          {t('finances.maintenanceCost', { cost: formatCurrency(calculateStadiumMaintenanceCost(stadium.capacity)) })}
          {' · '}
          {t('finances.stadiumCard.currentPrice', { price: formatCurrency(stadium.ticketPrice) })}
        </Typography>
        {upgradePending && (
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
            {t('finances.expandInProgress', {
              capacity: stadium.pendingCapacity === undefined ? undefined : formatNumber(stadium.pendingCapacity),
              date: formatDateTime(stadium.upgradeCompletesAt ?? ''),
            })}
          </Typography>
        )}
        {upgradeStadium.isSuccess && <Alert severity="success" sx={{ mb: 1 }}>{t('finances.expandSuccess')}</Alert>}
        {setTicketPrice.isSuccess && <Alert severity="success" sx={{ mb: 1 }}>{t('finances.ticketPriceSuccess')}</Alert>}

        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
          <Button variant="outlined" disabled={upgradePending} onClick={() => setDialog('expand')}>
            {t('finances.stadiumCard.expandButton')}
          </Button>
          <Button variant="outlined" onClick={() => setDialog('price')}>
            {t('finances.stadiumCard.priceButton')}
          </Button>
        </Stack>
      </CardContent>

      <ResponsiveDialog
        open={dialog === 'expand'}
        onClose={() => setDialog(null)}
        title={t('finances.stadiumCard.expandTitle')}
        subtitle={stadium.name}
        actions={
          <>
            <Button onClick={() => setDialog(null)}>{t('common.cancel')}</Button>
            <Button variant="contained" onClick={handleUpgrade} disabled={upgradeStadium.isPending || !seatsValid}>
              {t('finances.stadiumCard.confirmExpandButton', { cost: formatCurrency(expandCost ?? 0) })}
            </Button>
          </>
        }
      >
        <Stack spacing={2}>
          <Typography variant="body2" color="text.secondary">
            {t('finances.confirmExpand', { seats: seatsValid ? seatsNum : 0, cost: formatCurrency(expandCost ?? 0) })}
          </Typography>
          <NumberField
            label={t('finances.newSeats')}
            value={seats}
            onValueChange={setSeats}
            grouped
            error={!seatsValid}
            helperText={
              seatsValid
                ? t('finances.stadiumCard.costBefore', { cost: formatCurrency(expandCost ?? 0) })
                : t('finances.expandSeatsRange', { min: MIN_STADIUM_UPGRADE_SEATS, max: MAX_STADIUM_UPGRADE_SEATS })
            }
            fullWidth
          />
          {upgradeStadium.isError && <Alert severity="error">{t('finances.expandError')}</Alert>}
        </Stack>
      </ResponsiveDialog>

      <ResponsiveDialog
        open={dialog === 'price'}
        onClose={() => setDialog(null)}
        title={t('finances.stadiumCard.priceTitle')}
        subtitle={stadium.name}
        actions={
          <>
            <Button onClick={() => setDialog(null)}>{t('common.cancel')}</Button>
            <Button variant="contained" onClick={handlePriceSubmit} disabled={setTicketPrice.isPending || !priceValid}>
              {t('common.save')}
            </Button>
          </>
        }
      >
        <Stack spacing={2} component="form" onSubmit={handlePriceSubmit}>
          <NumberField
            label={t('finances.ticketPrice')}
            value={price}
            onValueChange={setPrice}
            currency
            error={!priceValid}
            helperText={
              priceValid
                ? t('finances.stadiumCard.priceHelper', { min: MIN_TICKET_PRICE, max: maxTicketPrice, recommended: recommendedTicketPrice })
                : t('finances.stadiumCard.priceRange', { min: MIN_TICKET_PRICE, max: maxTicketPrice })
            }
            fullWidth
          />
          <Button size="small" sx={{ alignSelf: 'flex-start' }} onClick={() => setPrice(String(recommendedTicketPrice))}>
            {t('finances.stadiumCard.useRecommended', { price: recommendedTicketPrice })}
          </Button>
          {setTicketPrice.isError && <Alert severity="error">{t('finances.ticketPriceError')}</Alert>}
        </Stack>
      </ResponsiveDialog>
    </Card>
  )
}
