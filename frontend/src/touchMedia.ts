// Interactive controls get a 44px touch target on touch devices and phone-width viewports
// (WCAG 2.5.5 / M3 guidance) without changing how they look on a desktop pointer - see theme.ts.
// Its own module so a component can tune a control's touch size without importing the theme.
export const TOUCH = '@media (pointer: coarse), (max-width: 599.95px)'
