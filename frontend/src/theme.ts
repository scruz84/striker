import { argbFromHex, hexFromArgb, themeFromSourceColor } from '@material/material-color-utilities'
import type { CSSProperties } from 'react'
import { createTheme } from '@mui/material/styles'
import type { PaletteColorOptions, Theme } from '@mui/material/styles'
import type { Scheme } from '@material/material-color-utilities'

// Material 3's color system starts from one seed color and derives every role (primary,
// secondary, tertiary, error, surface, outline, and their "container"/"on" pairs) from its hue -
// see https://m3.material.io/styles/color/system/how-the-system-works. The seed below is the
// app's existing brand green, so this is a restyle of the established identity onto real M3
// tonal-palette tokens, not a rebrand.
const SEED_COLOR = '#4CAF50'
const m3 = themeFromSourceColor(argbFromHex(SEED_COLOR))

function hex(argb: number): string {
  return hexFromArgb(argb)
}

// Blends `amount` of `tint` into `base` - used to approximate M3's tonal elevation (elevated
// surfaces take on a whisper of the primary color instead of relying on shadow alone).
function mixHex(base: string, tint: string, amount: number): string {
  const b = parseInt(base.slice(1), 16)
  const t = parseInt(tint.slice(1), 16)
  const br = (b >> 16) & 0xff
  const bg = (b >> 8) & 0xff
  const bb = b & 0xff
  const tr = (t >> 16) & 0xff
  const tg = (t >> 8) & 0xff
  const tb = t & 0xff
  const mix = (from: number, to: number) => Math.round(from + (to - from) * amount)
  return `#${[mix(br, tr), mix(bg, tg), mix(bb, tb)].map((v) => v.toString(16).padStart(2, '0')).join('')}`
}

declare module '@mui/material/styles' {
  // Small-print variants so chip/stat text never needs a hardcoded `fontSize` below the app's
  // 11px floor (see MobileLayout / #272).
  interface TypographyVariants {
    chipLabel: CSSProperties
    stat: CSSProperties
    pageTitle: CSSProperties
  }
  interface TypographyVariantsOptions {
    chipLabel?: CSSProperties
    stat?: CSSProperties
    pageTitle?: CSSProperties
  }
  interface Palette {
    tertiary: Palette['primary']
    primaryContainer: string
    onPrimaryContainer: string
    secondaryContainer: string
    onSecondaryContainer: string
    tertiaryContainer: string
    onTertiaryContainer: string
    surfaceVariant: string
    onSurfaceVariant: string
    outline: string
    outlineVariant: string
    inverseSurface: string
    inverseOnSurface: string
    inversePrimary: string
    // One-tone sequential scale (neutral -> primary) for player ratings, so low values do not
    // read as errors and the scale survives colour-blindness / grayscale (#283).
    rating: string[]
  }
  interface PaletteOptions {
    tertiary?: PaletteColorOptions
    primaryContainer?: string
    onPrimaryContainer?: string
    secondaryContainer?: string
    onSecondaryContainer?: string
    tertiaryContainer?: string
    onTertiaryContainer?: string
    surfaceVariant?: string
    onSurfaceVariant?: string
    outline?: string
    outlineVariant?: string
    inverseSurface?: string
    inverseOnSurface?: string
    inversePrimary?: string
    rating?: string[]
  }
}

declare module '@mui/material/Typography' {
  interface TypographyPropsVariantOverrides {
    chipLabel: true
    stat: true
    pageTitle: true
  }
}

function buildPalette(scheme: Scheme, mode: 'light' | 'dark') {
  return {
    mode,
    primary: { main: hex(scheme.primary), contrastText: hex(scheme.onPrimary) },
    secondary: { main: hex(scheme.secondary), contrastText: hex(scheme.onSecondary) },
    tertiary: { main: hex(scheme.tertiary), contrastText: hex(scheme.onTertiary) },
    error: { main: hex(scheme.error), contrastText: hex(scheme.onError) },
    background: { default: hex(scheme.background), paper: hex(scheme.surface) },
    text: { primary: hex(scheme.onBackground), secondary: hex(scheme.onSurfaceVariant) },
    divider: hex(scheme.outlineVariant),
    primaryContainer: hex(scheme.primaryContainer),
    onPrimaryContainer: hex(scheme.onPrimaryContainer),
    secondaryContainer: hex(scheme.secondaryContainer),
    onSecondaryContainer: hex(scheme.onSecondaryContainer),
    tertiaryContainer: hex(scheme.tertiaryContainer),
    onTertiaryContainer: hex(scheme.onTertiaryContainer),
    surfaceVariant: hex(scheme.surfaceVariant),
    onSurfaceVariant: hex(scheme.onSurfaceVariant),
    outline: hex(scheme.outline),
    outlineVariant: hex(scheme.outlineVariant),
    inverseSurface: hex(scheme.inverseSurface),
    inverseOnSurface: hex(scheme.inverseOnSurface),
    inversePrimary: hex(scheme.inversePrimary),
    rating: buildRatingScale(hex(scheme.outline), hex(scheme.primary), mode),
  }
}

// Five steps from the neutral outline tone up to the full primary - lightness is monotonic, so the
// steps stay distinguishable in grayscale.
function buildRatingScale(low: string, high: string, mode: 'light' | 'dark'): string[] {
  const start = mixHex(low, mode === 'light' ? '#000000' : '#ffffff', 0.15)
  const end = mixHex(high, mode === 'light' ? '#000000' : '#ffffff', mode === 'light' ? 0.25 : 0.1)
  return [0, 0.25, 0.5, 0.75, 1].map((step) => mixHex(start, end, step))
}

// The M3 type scale (Display/Headline/Title/Body/Label, each Large/Medium/Small) mapped onto
// MUI's variant names - see https://m3.material.io/styles/typography/type-scale-tokens. Sizes,
// line heights and tracking are M3's spec values; textTransform: 'none' on button/tab labels
// replaces MUI's default all-caps with M3's sentence case.
const typography = {
  fontFamily: '"Roboto", "Helvetica Neue", Arial, sans-serif',
  h1: { fontSize: '3.5625rem', lineHeight: '4rem', fontWeight: 400, letterSpacing: '-0.015625rem' },
  h2: { fontSize: '2.8125rem', lineHeight: '3.25rem', fontWeight: 400 },
  h3: { fontSize: '2.25rem', lineHeight: '2.75rem', fontWeight: 400 },
  h4: { fontSize: '1.75rem', lineHeight: '2.25rem', fontWeight: 400 },
  h5: { fontSize: '1.5rem', lineHeight: '2rem', fontWeight: 400 },
  h6: { fontSize: '1.375rem', lineHeight: '1.75rem', fontWeight: 400 },
  subtitle1: { fontSize: '1rem', lineHeight: '1.5rem', fontWeight: 500, letterSpacing: '0.009375rem' },
  subtitle2: { fontSize: '0.875rem', lineHeight: '1.25rem', fontWeight: 500, letterSpacing: '0.00625rem' },
  // Body tracking is tighter than the M3 spec (0.5px/0.25px): at phone widths the spec values make
  // Spanish paragraphs read as spaced-out.
  body1: { fontSize: '1rem', lineHeight: '1.5rem', fontWeight: 400, letterSpacing: '0.009375rem' },
  body2: { fontSize: '0.875rem', lineHeight: '1.25rem', fontWeight: 400, letterSpacing: '0.00625rem' },
  button: { fontSize: '0.875rem', lineHeight: '1.25rem', fontWeight: 500, letterSpacing: '0.00625rem', textTransform: 'none' as const },
  caption: { fontSize: '0.75rem', lineHeight: '1rem', fontWeight: 400, letterSpacing: '0.025rem' },
  overline: { fontSize: '0.6875rem', lineHeight: '1rem', fontWeight: 500, letterSpacing: '0.03125rem' },
  // Smallest text the app ever renders (11px floor).
  chipLabel: { fontSize: '0.6875rem', lineHeight: '1rem', fontWeight: 600, letterSpacing: '0.01em' },
  // Big figures: tabular so columns of numbers line up, Oswald for the sporty identity.
  stat: { fontFamily: '"Oswald", "Roboto", sans-serif', fontWeight: 600, fontVariantNumeric: 'tabular-nums' as const, lineHeight: 1.2 },
  // Page titles: one line at 360px, growing with the viewport (see PageHeader).
  pageTitle: { fontFamily: '"Oswald", "Roboto", sans-serif', fontWeight: 600, fontSize: '1.375rem', lineHeight: '1.75rem' },
}

// Interactive controls get a 44px touch target on touch devices and phone-width viewports
// (WCAG 2.5.5 / M3 guidance) without changing how they look on a desktop pointer.
const TOUCH = '@media (pointer: coarse), (max-width: 599.95px)'

// An invisible box centered on a control, used as its ::after to enlarge the tap area without
// changing the layout. Callers set the horizontal extent and the height.
const TOUCH_HIT_AREA = { content: '""', position: 'absolute', top: '50%', transform: 'translateY(-50%)' } as const

// M3's shape scale, applied per-component below: pill-shaped buttons/chips ("full"), medium
// (12dp) cards and menus, extra-large (28dp) dialogs - see
// https://m3.material.io/styles/shape/shape-scale-tokens.
function buildComponents(mode: 'light' | 'dark') {
  const tintAmount = mode === 'light' ? 0.05 : 0.1
  return {
    MuiCssBaseline: { styleOverrides: {} },
    MuiButton: {
      defaultProps: { disableElevation: true },
      styleOverrides: {
        root: ({ theme }: { theme: Theme }) => ({
          borderRadius: 9999,
          paddingLeft: 24,
          paddingRight: 24,
          [TOUCH]: { minHeight: 44 },
          // Disabled buttons stay legible (>= 3:1) instead of fading into tinted surfaces; the
          // reason belongs in helper text next to the button (#283).
          '&.Mui-disabled': {
            color: theme.palette.onSurfaceVariant,
            opacity: 0.72,
          },
          '&.MuiButton-contained.Mui-disabled': {
            backgroundColor: theme.palette.surfaceVariant,
            color: theme.palette.onSurfaceVariant,
          },
          '&.MuiButton-outlined.Mui-disabled': { borderColor: theme.palette.outline },
        }),
        sizeSmall: { paddingLeft: 16, paddingRight: 16 },
        textSizeSmall: { [TOUCH]: { minWidth: 44 } },
      },
    },
    MuiIconButton: {
      styleOverrides: {
        root: { [TOUCH]: { minWidth: 44, minHeight: 44 } },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          borderRadius: 9999,
          // Clickable chips keep their compact look but grow an invisible hit area at least 44px
          // tall (8px past each side, more for a short chip).
          '&.MuiChip-clickable, &.MuiChip-deletable': {
            position: 'relative',
            [TOUCH]: { '&::after': { ...TOUCH_HIT_AREA, left: -8, right: -8, height: 'max(calc(100% + 16px), 44px)' } },
          },
        },
        labelSmall: { fontSize: '0.6875rem' },
      },
    },
    MuiLink: {
      styleOverrides: {
        // A link that stands on its own in a layout (a team or player name in a list row, a
        // division link under a header) grows the same invisible 44px hit area. Links inside
        // running text (a <p>/<span>) don't: there the band would cover the lines above and below.
        root: {
          [TOUCH]: {
            'div > &, li > &, td > &': { position: 'relative', '&::after': { ...TOUCH_HIT_AREA, left: 0, right: 0, height: 'max(100%, 44px)' } },
          },
        },
        // Standalone text links (buttons rendered as links) get a tappable height too.
        button: { [TOUCH]: { minHeight: 44 } },
      },
    },
    MuiTab: {
      styleOverrides: { root: { minHeight: 48 } },
    },
    MuiListItemButton: {
      styleOverrides: { root: { [TOUCH]: { minHeight: 48 } } },
    },
    MuiMenuItem: {
      styleOverrides: { root: { [TOUCH]: { minHeight: 48 } } },
    },
    MuiSwitch: {
      styleOverrides: { root: { [TOUCH]: { '& .MuiSwitch-switchBase': { padding: 9 } } } },
    },
    MuiTextField: {
      // One TextField variant and height (40px with a mouse, 44px on touch - see MuiOutlinedInput)
      // across the app, with a label that is always visible (#282).
      defaultProps: { variant: 'outlined' as const, size: 'small' as const },
    },
    MuiFormControl: {
      defaultProps: { size: 'small' as const },
    },
    MuiSlider: {
      // MUI pads a slider to 40px on touch screens; the small variant's 2px rail leaves it at 42.
      styleOverrides: { root: { [TOUCH]: { '&:not(.MuiSlider-vertical)': { paddingTop: 21, paddingBottom: 21 } } } },
    },
    MuiToggleButton: {
      styleOverrides: { root: { [TOUCH]: { minHeight: 44 } } },
    },
    MuiPaper: {
      styleOverrides: {
        root: { backgroundImage: 'none' },
        rounded: { borderRadius: 12 },
        outlined: ({ theme }: { theme: Theme }) => ({
          borderColor: theme.palette.outlineVariant,
          boxShadow: 'none',
        }),
        elevation1: ({ theme }: { theme: Theme }) => ({
          boxShadow: 'none',
          backgroundColor: mixHex(theme.palette.background.paper, theme.palette.primary.main, tintAmount),
        }),
      },
    },
    MuiDialog: {
      styleOverrides: { paper: { borderRadius: 28 } },
    },
    MuiDialogTitle: {
      styleOverrides: { root: { overflowWrap: 'anywhere' as const } },
    },
    MuiOutlinedInput: {
      // 44px tall on touch screens, like every other tappable control (#283); small fields stay
      // compact with a mouse.
      styleOverrides: { root: { borderRadius: 4, [TOUCH]: { minHeight: 44 } } },
    },
    MuiAppBar: {
      styleOverrides: { root: { boxShadow: 'none' } },
    },
    MuiTableContainer: {
      styleOverrides: { root: { borderRadius: 12 } },
    },
    MuiAlert: {
      styleOverrides: { root: { borderRadius: 12 } },
    },
  }
}

export const lightTheme = createTheme({
  palette: buildPalette(m3.schemes.light, 'light'),
  shape: { borderRadius: 12 },
  typography,
  components: buildComponents('light'),
})

export const darkTheme = createTheme({
  palette: buildPalette(m3.schemes.dark, 'dark'),
  shape: { borderRadius: 12 },
  typography,
  components: buildComponents('dark'),
})
