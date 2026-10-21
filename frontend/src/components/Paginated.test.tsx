import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Paginated } from './Paginated'

const rows = Array.from({ length: 60 }, (_, i) => i + 1)

function renderList(resetKey: string) {
  return (
    <Paginated rows={rows} resetKey={resetKey}>
      {(pageRows) => <div data-testid="rows">{pageRows.join(',')}</div>}
    </Paginated>
  )
}

describe('Paginated', () => {
  it('renders only the first page and pages forward', () => {
    render(renderList('a'))
    expect(screen.getByTestId('rows').textContent?.split(',')).toHaveLength(20)
    expect(screen.getByText('1–20 of 60')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /next page/i }))
    expect(screen.getByText('21–40 of 60')).toBeInTheDocument()
    expect(screen.getByTestId('rows').textContent?.startsWith('21,')).toBe(true)
  })

  it('goes back to the first page when the reset key changes', () => {
    const { rerender } = render(renderList('a'))
    fireEvent.click(screen.getByRole('button', { name: /next page/i }))
    rerender(renderList('b'))
    expect(screen.getByText('1–20 of 60')).toBeInTheDocument()
  })

  it('hides the pager when everything fits on one page', () => {
    render(
      <Paginated rows={[1, 2, 3]}>{(pageRows) => <div>{pageRows.join(',')}</div>}</Paginated>,
    )
    expect(screen.getByText('1,2,3')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /next page/i })).not.toBeInTheDocument()
  })
})
