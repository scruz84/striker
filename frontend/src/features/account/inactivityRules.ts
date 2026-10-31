// The inactivity rules as the backend applies them (common.InactivityReleaseDays /
// AccountErasureDays). The Account page reads them from the API; screens without that call
// (onboarding's "your club was released" notice) use these constants.
export const InactivityReleaseDays = 60
export const AccountErasureDays = 365
