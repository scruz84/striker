import { argbFromHex, DynamicScheme, Hct, hexFromArgb, TonalPalette, Variant } from '@material/material-color-utilities'
import type { CSSProperties } from 'react'
import { alpha, createTheme } from '@mui/material/styles'
import { TOUCH } from './touchMedia'
import type { PaletteColorOptions, Theme } from '@mui/material/styles'

// Material 3's color system starts from one seed color and derives every role (primary,
// secondary, tertiary, error, the surface and surface-container levels, outline, and their
// "container"/"on" pairs) from its hue - see https://m3.material.io/styles/color/system/how-the-system-works.
// The seed is the app's brand green. The scheme is an M3 DynamicScheme (tonal-spot role tones,
// including the surface containers) built over the same core palettes the legacy `Scheme` used:
// the preset SchemeTonalSpot caps primary chroma at 36 and would turn the brand green into an
// olive (#3b6939 instead of #006e1c). theme.test.ts pins the resulting colors and their contrast.
const SEED_COLOR = '#4CAF50'
const seedHct = Hct.fromInt(argbFromHex(SEED_COLOR))

// contrastLevel 0 is M3's standard scheme; 1 is its high-contrast scheme, used when the system
// asks for more contrast (prefers-contrast: more - see ThemeModeProvider).
function buildScheme(isDark: boolean, contrastLevel = 0): DynamicScheme {
  const { hue, chroma } = seedHct
  return new DynamicScheme({
    sourceColorHct: seedHct,
    variant: Variant.TONAL_SPOT,
    contrastLevel,
    isDark,
    primaryPalette: TonalPalette.fromHueAndChroma(hue, Math.max(48, chroma)),
    secondaryPalette: TonalPalette.fromHueAndChroma(hue, 16),
    tertiaryPalette: TonalPalette.fromHueAndChroma(hue + 60, 24),
    neutralPalette: TonalPalette.fromHueAndChroma(hue, 4),
    neutralVariantPalette: TonalPalette.fromHueAndChroma(hue, 8),
  })
}

function hex(argb: number): string {
  return hexFromArgb(argb)
}

// Blends `amount` of `tint` into `base` - used to build the rating scale.
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
    errorContainer: string
    onErrorContainer: string
    surfaceVariant: string
    onSurfaceVariant: string
    outline: string
    outlineVariant: string
    inverseSurface: string
    inverseOnSurface: string
    inversePrimary: string
    onSurface: string
    surfaceDim: string
    surfaceBright: string
    surfaceContainerLowest: string
    surfaceContainerLow: string
    surfaceContainer: string
    surfaceContainerHigh: string
    surfaceContainerHighest: string
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
    errorContainer?: string
    onErrorContainer?: string
    surfaceVariant?: string
    onSurfaceVariant?: string
    outline?: string
    outlineVariant?: string
    inverseSurface?: string
    inverseOnSurface?: string
    inversePrimary?: string
    onSurface?: string
    surfaceDim?: string
    surfaceBright?: string
    surfaceContainerLowest?: string
    surfaceContainerLow?: string
    surfaceContainer?: string
    surfaceContainerHigh?: string
    surfaceContainerHighest?: string
    rating?: string[]
  }
}

declare module '@mui/material/Button' {
  // M3's tonal button: between filled (contained) and outlined in emphasis.
  interface ButtonPropsVariantOverrides {
    tonal: true
  }
}

declare module '@mui/material/Paper' {
  // M3 card styles, used as <Card variant="elevated" | "filled" | "outlined"> (Card takes
  // Paper's variants). "outlined" is MUI's own.
  interface PaperPropsVariantOverrides {
    elevated: true
    filled: true
  }
}

declare module '@mui/material/Typography' {
  interface TypographyPropsVariantOverrides {
    chipLabel: true
    stat: true
    pageTitle: true
  }
}

function buildPalette(scheme: DynamicScheme, mode: 'light' | 'dark') {
  return {
    mode,
    primary: { main: hex(scheme.primary), contrastText: hex(scheme.onPrimary) },
    secondary: { main: hex(scheme.secondary), contrastText: hex(scheme.onSecondary) },
    tertiary: { main: hex(scheme.tertiary), contrastText: hex(scheme.onTertiary) },
    error: { main: hex(scheme.error), contrastText: hex(scheme.onError) },
    // Outlined panels sit on the page colour; raised ones get surfaceContainerLow (MuiPaper below).
    background: { default: hex(scheme.surface), paper: hex(scheme.surface) },
    text: { primary: hex(scheme.onSurface), secondary: hex(scheme.onSurfaceVariant) },
    divider: hex(scheme.outlineVariant),
    primaryContainer: hex(scheme.primaryContainer),
    onPrimaryContainer: hex(scheme.onPrimaryContainer),
    secondaryContainer: hex(scheme.secondaryContainer),
    onSecondaryContainer: hex(scheme.onSecondaryContainer),
    tertiaryContainer: hex(scheme.tertiaryContainer),
    onTertiaryContainer: hex(scheme.onTertiaryContainer),
    errorContainer: hex(scheme.errorContainer),
    onErrorContainer: hex(scheme.onErrorContainer),
    // M3 deprecated surfaceVariant in favour of surfaceContainerHighest; kept for the disabled
    // button style below.
    surfaceVariant: hex(scheme.surfaceContainerHighest),
    onSurfaceVariant: hex(scheme.onSurfaceVariant),
    outline: hex(scheme.outline),
    outlineVariant: hex(scheme.outlineVariant),
    inverseSurface: hex(scheme.inverseSurface),
    inverseOnSurface: hex(scheme.inverseOnSurface),
    inversePrimary: hex(scheme.inversePrimary),
    onSurface: hex(scheme.onSurface),
    surfaceDim: hex(scheme.surfaceDim),
    surfaceBright: hex(scheme.surfaceBright),
    surfaceContainerLowest: hex(scheme.surfaceContainerLowest),
    surfaceContainerLow: hex(scheme.surfaceContainerLow),
    surfaceContainer: hex(scheme.surfaceContainer),
    surfaceContainerHigh: hex(scheme.surfaceContainerHigh),
    surfaceContainerHighest: hex(scheme.surfaceContainerHighest),
    rating: buildRatingScale(hex(scheme.outline), hex(scheme.primary), mode),
    // M3 state layers: hover 8%, focus and pressed 10% of the content colour (MUI defaults to
    // 4% / 12%). Text, outlined and icon buttons, list items and menus all derive from these.
    action: {
      active: hex(scheme.onSurfaceVariant),
      hover: alpha(hex(scheme.onSurface), 0.08),
      hoverOpacity: 0.08,
      focus: alpha(hex(scheme.onSurface), 0.1),
      focusOpacity: 0.1,
      activatedOpacity: 0.1,
    },
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


// An invisible box centered on a control, used as its ::after to enlarge the tap area without
// changing the layout. Callers set the horizontal extent and the height.
const TOUCH_HIT_AREA = { content: '""', position: 'absolute', top: '50%', transform: 'translateY(-50%)' } as const

// M3 components on top of MUI, from the theme alone so pages need no changes (see the ui-theme
// spec): a surface-coloured top app bar, four button emphasis levels (contained = filled,
// variant="tonal", outlined, text), 8dp chips, segmented buttons (ToggleButtonGroup), card
// styles (variant="elevated" | "filled" | "outlined"), and M3 tabs, switches, linear progress,
// menus and dialogs. Shapes follow the M3 scale - full (pill) buttons, small (8dp) chips, medium
// (12dp) cards, extra-small (4dp) menus, extra-large (28dp) dialogs - see
// https://m3.material.io/styles/shape/shape-scale-tokens.
// A level-1 M3 elevation shadow (key + ambient), for elevated cards.
const ELEVATION_1 = '0 1px 2px rgba(0, 0, 0, 0.3), 0 1px 3px 1px rgba(0, 0, 0, 0.15)'

function buildComponents() {
  return {
    MuiCssBaseline: {
      styleOverrides: {
        // Honour the system's "reduce motion" setting: transitions and animations (slides,
        // ripples, the pulsing LIVE chip) finish at once instead of moving.
        '@media (prefers-reduced-motion: reduce)': {
          '*, *::before, *::after': {
            animationDuration: '0.01ms !important',
            animationIterationCount: '1 !important',
            transitionDuration: '0.01ms !important',
            scrollBehavior: 'auto !important',
          },
        },
      },
    },
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
        // M3 buttons are 40dp tall; the compact size the app uses in dense rows keeps MUI's.
        sizeMedium: { minHeight: 40 },
        sizeLarge: { minHeight: 40 },
        sizeSmall: { paddingLeft: 16, paddingRight: 16 },
        textSizeSmall: { [TOUCH]: { minWidth: 44 } },
      },
      variants: [
        {
          props: { variant: 'tonal' as const },
          style: ({ theme }: { theme: Theme }) => ({
            backgroundColor: theme.palette.secondaryContainer,
            color: theme.palette.onSecondaryContainer,
            // M3 state layer: 8% of the content colour on hover.
            '&:hover': { backgroundColor: `color-mix(in srgb, ${theme.palette.onSecondaryContainer} 8%, ${theme.palette.secondaryContainer})` },
          }),
        },
      ],
    },
    MuiIconButton: {
      styleOverrides: {
        root: { [TOUCH]: { minWidth: 44, minHeight: 44 } },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          borderRadius: 8,
          height: 32,
          // Clickable chips keep their compact look but grow an invisible hit area at least 44px
          // tall (8px past each side, more for a short chip).
          '&.MuiChip-clickable, &.MuiChip-deletable': {
            position: 'relative',
            [TOUCH]: { '&::after': { ...TOUCH_HIT_AREA, left: -8, right: -8, height: 'max(calc(100% + 16px), 44px)' } },
          },
        },
        sizeSmall: { height: 24 },
        labelSmall: { fontSize: '0.6875rem' },
        outlined: ({ theme }: { theme: Theme }) => ({
          '&.MuiChip-colorDefault': { borderColor: theme.palette.outlineVariant },
        }),
        // A filled primary chip marks something selected ("You", the chosen sponsor deal...): M3's
        // selected filter chip colours. Status colours (success, error...) keep MUI's palette.
        filled: ({ theme }: { theme: Theme }) => ({
          '&.MuiChip-colorPrimary': {
            backgroundColor: theme.palette.secondaryContainer,
            color: theme.palette.onSecondaryContainer,
            '& .MuiChip-icon': { color: 'inherit' },
          },
        }),
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
    MuiTabs: {
      styleOverrides: { indicator: { height: 3, borderRadius: '3px 3px 0 0' } },
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
      // M3 switch: a 52x32 track inside a 64x48 tap area; the thumb is 16px (outline colour) off
      // and 24px (onPrimary) on, 16px from the track's end either way.
      styleOverrides: {
        root: { width: 68, height: 48, padding: 8 },
        switchBase: ({ theme }: { theme: Theme }) => ({
          top: 8,
          left: 8,
          padding: 8,
          color: theme.palette.outline,
          '&.Mui-checked': { transform: 'translateX(20px)', color: theme.palette.primary.contrastText },
          '&.Mui-checked + .MuiSwitch-track': { opacity: 1, backgroundColor: theme.palette.primary.main, borderColor: theme.palette.primary.main },
          '&.Mui-checked .MuiSwitch-thumb': { width: 24, height: 24, margin: -4 },
          '&.Mui-disabled + .MuiSwitch-track': { opacity: 0.12 },
        }),
        thumb: { width: 16, height: 16, boxShadow: 'none' },
        track: ({ theme }: { theme: Theme }) => ({
          boxSizing: 'border-box' as const,
          height: 32,
          borderRadius: 16,
          opacity: 1,
          backgroundColor: theme.palette.surfaceContainerHighest,
          border: `2px solid ${theme.palette.outline}`,
        }),
      },
    },
    MuiLinearProgress: {
      styleOverrides: {
        root: { height: 4, borderRadius: 2 },
        colorPrimary: ({ theme }: { theme: Theme }) => ({ backgroundColor: theme.palette.secondaryContainer }),
        bar: { borderRadius: 2 },
      },
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
    // Single-choice button groups render as M3 segmented buttons: a pill outline, 1px dividers
    // and the selected segment in secondaryContainer. A page that compacts a group with sx keeps
    // its own height.
    MuiToggleButtonGroup: {
      styleOverrides: {
        grouped: ({ theme }: { theme: Theme }) => ({ borderColor: theme.palette.outline }),
        firstButton: { borderTopLeftRadius: 9999, borderBottomLeftRadius: 9999 },
        lastButton: { borderTopRightRadius: 9999, borderBottomRightRadius: 9999 },
      },
    },
    MuiToggleButton: {
      styleOverrides: {
        root: ({ theme }: { theme: Theme }) => ({
          minHeight: 40,
          borderRadius: 9999,
          textTransform: 'none' as const,
          color: theme.palette.onSurface,
          borderColor: theme.palette.outline,
          [TOUCH]: { minHeight: 44 },
          '&.Mui-selected, &.Mui-selected:hover': {
            backgroundColor: theme.palette.secondaryContainer,
            color: theme.palette.onSecondaryContainer,
          },
        }),
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: { backgroundImage: 'none' },
        rounded: { borderRadius: 12 },
        outlined: ({ theme }: { theme: Theme }) => ({
          borderColor: theme.palette.outlineVariant,
          boxShadow: 'none',
        }),
        // A raised panel sits one surface-container level up instead of casting a shadow or
        // blending in the primary colour.
        elevation1: ({ theme }: { theme: Theme }) => ({
          boxShadow: 'none',
          backgroundColor: theme.palette.surfaceContainerLow,
        }),
      },
    },
    MuiCard: {
      variants: [
        {
          props: { variant: 'elevated' as const },
          style: ({ theme }: { theme: Theme }) => ({ backgroundColor: theme.palette.surfaceContainerLow, boxShadow: ELEVATION_1 }),
        },
        {
          props: { variant: 'filled' as const },
          style: ({ theme }: { theme: Theme }) => ({ backgroundColor: theme.palette.surfaceContainerHighest, boxShadow: 'none' }),
        },
      ],
    },
    MuiMenu: {
      styleOverrides: {
        paper: ({ theme }: { theme: Theme }) => ({ backgroundColor: theme.palette.surfaceContainer, borderRadius: 4 }),
      },
    },
    MuiPopover: {
      styleOverrides: {
        paper: ({ theme }: { theme: Theme }) => ({ backgroundColor: theme.palette.surfaceContainer, borderRadius: 4 }),
      },
    },
    MuiDialog: {
      styleOverrides: {
        paper: ({ theme }: { theme: Theme }) => ({ borderRadius: 28, backgroundColor: theme.palette.surfaceContainerHigh }),
        // Phones get full-screen dialogs (ResponsiveDialog): square, on the page surface like
        // the app bar ResponsiveDialog draws on top of them.
        paperFullScreen: ({ theme }: { theme: Theme }) => ({ borderRadius: 0, backgroundColor: theme.palette.background.default }),
      },
    },
    MuiDialogTitle: {
      styleOverrides: { root: { overflowWrap: 'anywhere' as const } },
    },
    MuiOutlinedInput: {
      // 44px tall on touch screens, like every other tappable control (#283); small fields stay
      // compact with a mouse.
      styleOverrides: { root: { borderRadius: 4, [TOUCH]: { minHeight: 44 } } },
    },
    // M3 top app bar: the page surface, onSurface content, no shadow. Header.tsx switches it to
    // surfaceContainer while content is scrolled under it.
    MuiAppBar: {
      defaultProps: { color: 'default' as const },
      styleOverrides: {
        root: { boxShadow: 'none' },
        colorDefault: ({ theme }: { theme: Theme }) => ({
          backgroundColor: theme.palette.background.default,
          color: theme.palette.onSurface,
          transition: 'background-color 200ms',
        }),
      },
    },
    // M3 navigation bar (BottomNav, the app's only BottomNavigation): 80px on surfaceContainer,
    // a 64x32 pill behind the current entry's icon, 12px labels (MUI would grow the current one
    // to 14px). See the app-navigation spec.
    MuiBottomNavigation: {
      styleOverrides: {
        root: ({ theme }: { theme: Theme }) => ({ height: 80, backgroundColor: theme.palette.surfaceContainer }),
      },
    },
    MuiBottomNavigationAction: {
      styleOverrides: {
        root: ({ theme }: { theme: Theme }) => ({
          minWidth: 0,
          gap: 4,
          paddingTop: 12,
          justifyContent: 'flex-start',
          color: theme.palette.onSurfaceVariant,
          '& .MuiSvgIcon-root': {
            width: 64,
            height: 32,
            padding: '4px 20px',
            boxSizing: 'border-box' as const,
            borderRadius: 16,
            transition: 'background-color 200ms',
          },
          '&.Mui-selected': { color: theme.palette.onSurface },
          '&.Mui-selected .MuiSvgIcon-root': {
            backgroundColor: theme.palette.secondaryContainer,
            color: theme.palette.onSecondaryContainer,
          },
        }),
        label: {
          fontSize: '0.75rem',
          lineHeight: '1rem',
          fontWeight: 500,
          whiteSpace: 'nowrap' as const,
          '&.Mui-selected': { fontSize: '0.75rem', fontWeight: 700 },
        },
      },
    },
    MuiTableContainer: {
      styleOverrides: { root: { borderRadius: 12 } },
    },
    MuiAlert: {
      styleOverrides: { root: { borderRadius: 12 } },
    },
  }
}

// M3 motion: the standard and emphasized easing curves and the duration tokens (short4 200ms for
// exits, medium2 300ms for most changes, medium4 400ms for entering surfaces) in place of MUI's
// Material 2 values - see https://m3.material.io/styles/motion/easing-and-duration/tokens-specs.
export const M3_TRANSITIONS = {
  easing: {
    easeInOut: 'cubic-bezier(0.2, 0, 0, 1)', // standard
    easeOut: 'cubic-bezier(0.05, 0.7, 0.1, 1)', // emphasized decelerate (entering)
    easeIn: 'cubic-bezier(0.3, 0, 0.8, 0.15)', // emphasized accelerate (exiting)
    sharp: 'cubic-bezier(0.3, 0, 1, 1)', // standard accelerate
  },
  duration: {
    shortest: 100,
    shorter: 150,
    short: 200,
    standard: 300,
    complex: 400,
    enteringScreen: 400,
    leavingScreen: 200,
  },
}

/** The app theme for a mode, in M3's standard scheme or, when the system asks for more contrast,
 * its high-contrast scheme. */
export function buildTheme(mode: 'light' | 'dark', highContrast = false): Theme {
  return createTheme({
    palette: buildPalette(buildScheme(mode === 'dark', highContrast ? 1 : 0), mode),
    shape: { borderRadius: 12 },
    typography,
    transitions: M3_TRANSITIONS,
    components: buildComponents(),
  })
}

export const lightTheme = buildTheme('light')
export const darkTheme = buildTheme('dark')
export const highContrastLightTheme = buildTheme('light', true)
export const highContrastDarkTheme = buildTheme('dark', true)
