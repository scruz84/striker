# notifications Specification

## Purpose

How in-game events (results, transfers, progression, awards, call-ups, fanbase…) reach a
manager: the in-app feed, localized rendering, and optional external delivery by Web Push.

## Requirements

### Requirement: In-app notification feed
Every notification SHALL be stored as a row for the recipient (optionally with a link) and shown in the bell menu, which also polls the unread count. Notifications for an unowned team MUST be skipped silently.

#### Scenario: Owned team sells a player
- **WHEN** an owner's player is sold
- **THEN** a notification row is created and appears in their bell menu

#### Scenario: Unowned team
- **WHEN** an event concerns a team nobody owns
- **THEN** no notification is created

### Requirement: Localizable messages
Every notification SHALL be built as a message with English text, an i18n key and its parameters, all stored. The frontend MUST render the key in the viewer's language when present and fall back to the English text otherwise.

#### Scenario: Spanish viewer
- **WHEN** a Spanish-speaking user opens a notification that has a key
- **THEN** it is rendered from `notifications.messages.*` in `es.json`

#### Scenario: Old row
- **WHEN** a notification predates keys
- **THEN** its stored English text is shown

#### Scenario: Adding a notification type
- **WHEN** a developer adds a notification type
- **THEN** they add the key and params at the call site, the `notifications.messages.*` entry in `en.json` and `es.json`, and the Spanish template in `common/notificationtranslations.go`

### Requirement: Notification preferences
Each user SHALL have email and push delivery preferences, both off by default. The in-app notification MUST be created regardless of them.

#### Scenario: Both off
- **WHEN** a user has both preferences off
- **THEN** only the in-app notification is created

### Requirement: Localized external delivery
External delivery SHALL be best-effort: it MUST never fail the in-app notification. It renders in the user's stored language (set when they switch language) using the backend's Spanish templates for `es`, and English otherwise or for an unknown key.

#### Scenario: Spanish user with push on
- **WHEN** a notification with a known key is sent to a user whose language is `es`
- **THEN** the push text is the Spanish rendering

#### Scenario: Unknown key
- **WHEN** the key has no Spanish template
- **THEN** the English text is sent

### Requirement: Web Push channel
When `VAPID_PUBLIC_KEY` and `VAPID_PRIVATE_KEY` are configured, both the API and engine processes SHALL deliver push notifications to every subscribed device of the user, dropping a subscription its push service reports gone (404/410). Otherwise delivery is only logged. Email is always log-only.

#### Scenario: VAPID keys missing
- **WHEN** the keys are not set
- **THEN** push notifications are logged and never sent, with a startup warning

#### Scenario: Expired subscription
- **WHEN** a push service answers 410 for a subscription
- **THEN** that subscription is deleted

### Requirement: Browser subscription
The frontend SHALL only turn push on after the browser grants permission and the push subscription succeeds; the service worker displays notifications and opens or focuses the app when one is clicked.

#### Scenario: Permission denied
- **WHEN** a user toggles push on but denies browser permission
- **THEN** push stays off

## Code map

- Storage and delivery: `backend/common/notifications.go` (`CreateNotification*`,
  `NotifyTeamOwner*`, `deliverExternalNotifications`), `common/notificationprefs.go`,
  `common/notifier.go` (`LogNotifier`, `DefaultNotifier`), `common/webpushnotifier.go`,
  `common/pushsubscriptions.go`, `common/notificationtranslations.go`.
- Frontend: `NotificationBell.tsx`, `LanguageSwitcher.tsx` (`PUT /v1/user/language`),
  `src/pushNotifications.ts`, `public/sw.js`, `src/i18n/locales/{en,es}.json`.
