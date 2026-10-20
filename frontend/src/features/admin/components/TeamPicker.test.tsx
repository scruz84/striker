import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '../../../i18n'
import { TeamPicker, type PickedTeam } from './TeamPicker'

const useSearch = vi.hoisted(() => vi.fn())
vi.mock('../../../api/hooks', () => ({ useSearch }))

const betis = { id: 12, name: 'Real Betis', country: 'es', divisionName: 'Primera' }
const bilbao = { id: 30, name: 'Athletic Bilbao', country: 'es', divisionName: 'Primera' }

// A real user types into a focused field; MUI's Autocomplete resets an unfocused input, so focus it first.
function type(input: HTMLElement, text: string) {
  fireEvent.focus(input)
  fireEvent.change(input, { target: { value: text } })
}

function renderPicker(onChange = vi.fn(), value: PickedTeam | null = null) {
  render(<TeamPicker label="Team" value={value} onChange={onChange} />)
  return { onChange, input: screen.getByRole('combobox') }
}

describe('TeamPicker', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useSearch.mockImplementation((query: string) => ({
      data: { teams: query.toLowerCase().includes('b') ? [betis, bilbao] : [], players: [], coaches: [], stadiums: [] },
      isFetching: false,
    }))
  })

  it('searches by name (after the typing pauses) and lists the matches with their country and division', async () => {
    const { input } = renderPicker()
    type(input, 'bet')

    expect(await screen.findByText('Real Betis (#12)')).toBeInTheDocument()
    expect(screen.getByText('Athletic Bilbao (#30)')).toBeInTheDocument()
    expect(screen.getAllByText('ES · Primera')).toHaveLength(2)
    // the search was asked about the typed text, not once per keystroke before the pause
    expect(useSearch).toHaveBeenLastCalledWith('bet')
  })

  it('reports the chosen team with the id the API needs', async () => {
    const { input, onChange } = renderPicker()
    type(input, 'bet')
    fireEvent.click(await screen.findByText('Real Betis (#12)'))

    expect(onChange).toHaveBeenCalledWith({ id: 12, name: 'Real Betis', label: 'Real Betis (#12)', detail: 'ES · Primera' })
  })

  it('still accepts a bare team id, so an admin who only has the number is not blocked', async () => {
    const { input, onChange } = renderPicker()
    type(input, '42')
    fireEvent.click(await screen.findByText(i18n.t('admin.teamPicker.teamNumber', { id: 42 })))

    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ id: 42 }))
  })

  it('does not duplicate a team when the typed id is already among the results', async () => {
    useSearch.mockImplementation(() => ({ data: { teams: [betis], players: [], coaches: [], stadiums: [] }, isFetching: false }))
    const { input } = renderPicker()
    type(input, '12')

    expect(await screen.findByText('Real Betis (#12)')).toBeInTheDocument()
    expect(screen.queryByText(i18n.t('admin.teamPicker.teamNumber', { id: 12 }))).not.toBeInTheDocument()
  })

  it('says so when nothing matches', async () => {
    const { input } = renderPicker()
    type(input, 'zzz')
    await waitFor(() => expect(screen.getByText(i18n.t('admin.teamPicker.noResults'))).toBeInTheDocument())
  })

  it('shows the chosen team in the field', () => {
    renderPicker(vi.fn(), { id: 12, name: 'Real Betis', label: 'Real Betis (#12)', detail: '' })
    expect(screen.getByRole('combobox')).toHaveValue('Real Betis (#12)')
  })
})
