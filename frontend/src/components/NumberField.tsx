import type { ReactNode } from 'react'
import InputAdornment from '@mui/material/InputAdornment'
import TextField, { type TextFieldProps } from '@mui/material/TextField'
import { currencySymbol, formatNumber } from '../utils/formatCurrency'

type BaseProps = Omit<TextFieldProps, 'value' | 'onChange' | 'type' | 'variant'>

interface NumberFieldProps extends BaseProps {
  /** Digits only ("250000") - empty string for no value. */
  value: string
  onValueChange: (digits: string) => void
  /** Show the game currency as a prefix/suffix adornment. */
  currency?: boolean
  /** Group thousands according to the UI language ("250.000"). On for money by default. */
  grouped?: boolean
  /** Text after the value ("%", "min"). */
  suffix?: ReactNode
  min?: number
  max?: number
}

/** Integer input for phones: opens the numeric keypad (`inputMode="numeric"`), accepts digits
 * only, and shows thousands separators for the active language - so `250000` reads as `250.000`.
 * Always labelled (never placeholder-only). The value it reports is the plain digit string. */
export function NumberField({
  value,
  onValueChange,
  currency = false,
  grouped = currency,
  suffix,
  min,
  max,
  slotProps,
  ...rest
}: NumberFieldProps) {
  const display = grouped && value !== '' ? formatNumber(Number(value)) : value

  const adornment = currency ? (
    <InputAdornment position="end">{currencySymbol()}</InputAdornment>
  ) : suffix ? (
    <InputAdornment position="end">{suffix}</InputAdornment>
  ) : undefined

  return (
    <TextField
      {...rest}
      value={display}
      onChange={(event) => {
        let digits = event.target.value.replace(/\D/g, '').replace(/^0+(?=\d)/, '')
        if (digits !== '' && max !== undefined && Number(digits) > max) {
          digits = String(max)
        }
        onValueChange(digits)
      }}
      slotProps={{
        ...slotProps,
        htmlInput: {
          inputMode: 'numeric',
          pattern: '[0-9]*',
          min,
          max,
          ...(slotProps?.htmlInput as object | undefined),
        },
        input: { endAdornment: adornment, ...(slotProps?.input as object | undefined) },
      }}
    />
  )
}
