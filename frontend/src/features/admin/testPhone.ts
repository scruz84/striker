// jsdom has no matchMedia, so MUI's useMediaQuery always reports "no match" (a desktop). This
// installs a stub that matches every query when `phone` is true - enough to flip useIsPhone, which
// only asks about the `sm` breakpoint.
export function setPhone(phone: boolean) {
  window.matchMedia = ((query: string) => ({
    matches: phone,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })) as typeof window.matchMedia
}
