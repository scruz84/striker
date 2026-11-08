import { render, screen } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import AccountBalanceWalletOutlinedIcon from '@mui/icons-material/AccountBalanceWalletOutlined'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { lightTheme } from '../../theme'
import { StatTile } from './StatTile'

const rgb = (hex: string) => `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`

function renderTile(props: Partial<Parameters<typeof StatTile>[0]> = {}) {
  render(
    <ThemeProvider theme={lightTheme}>
      <MemoryRouter>
        <StatTile icon={<AccountBalanceWalletOutlinedIcon />} label="Budget" value="€400k" to="/dashboard/finances" {...props} />
      </MemoryRouter>
    </ThemeProvider>,
  )
}

// ui-theme "Stat tiles".
describe('StatTile', () => {
  it('is a filled card that links as a whole, with a quiet label (Dashboard KPIs)', () => {
    renderTile()
    const link = screen.getByRole('link', { name: /Budget/ })
    expect(link).toHaveAttribute('href', '/dashboard/finances')
    expect(link).toHaveTextContent('€400k')
    expect(getComputedStyle(link.closest('.MuiCard-root')!).backgroundColor).toBe(rgb('#e2e3dd'))
    expect(getComputedStyle(screen.getByText('Budget')).color).toBe(rgb('#424940'))
    expect(getComputedStyle(screen.getByText('€400k')).color).toBe(rgb('#1a1c19'))
  })

  it('keeps a meaning colour on the value (Negative budget)', () => {
    renderTile({ value: '-€20k', valueColor: lightTheme.palette.error.main })
    expect(getComputedStyle(screen.getByText('-€20k')).color).toBe(rgb(lightTheme.palette.error.main))
  })

  it('is not a link without a destination', () => {
    renderTile({ to: undefined })
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    expect(screen.getByText('€400k')).toBeInTheDocument()
  })
})
