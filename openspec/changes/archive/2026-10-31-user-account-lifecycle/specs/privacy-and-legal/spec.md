# Spec Delta

## Purpose

What a public deployment must disclose to and enforce for its users: privacy policy, terms and
legal notice with an operator-configured identity, minimum age and terms acceptance, data
minimisation in public APIs, and storage limited to what is strictly necessary.

## ADDED Requirements

### Requirement: Privacy policy and terms pages
The app SHALL serve a Privacy Policy at `/privacy` and Terms of Use at `/terms`, in English and Spanish, readable without signing in, each showing its version and effective date. The login page, the help page, the navigation drawer and the Account page MUST link to both.

#### Scenario: Visitor reads the policy
- **WHEN** a visitor who is not signed in opens `/privacy` with the language set to Spanish
- **THEN** they see the Spanish privacy policy and its version

### Requirement: Privacy policy data and retention
The Privacy Policy SHALL state the controller and a contact address for privacy requests, the data collected (Google name and email, chosen country and language, activity times, push subscriptions, game data), the purposes and legal bases, the recipients (Google sign-in, hosting, browser push services) and the retention periods.

#### Scenario: Retention section
- **WHEN** a user reads the retention section
- **THEN** it states that a club is released after 60 days without activity, an account is erased after 365 days, and logs and backups are kept at most 30 days

### Requirement: Privacy policy rights and storage
The Privacy Policy SHALL explain the user's rights and how to exercise them, the right to complain to the supervisory authority (AEPD), and every cookie and browser-storage item the app uses with its purpose.

#### Scenario: Rights section
- **WHEN** a user reads the rights section
- **THEN** it explains that the Account page offers data download and account deletion, and gives the contact address for any other request

### Requirement: Age and terms acceptance at sign-up
A new user SHALL confirm they are at least 16 years old and accept the Terms and Privacy Policy before a club is assigned. The acceptance time and the policy version MUST be stored with the account.

#### Scenario: Box not ticked
- **WHEN** a new user picks a country without ticking the age and terms box
- **THEN** no club is assigned and the onboarding page asks them to tick it

#### Scenario: Acceptance recorded
- **WHEN** a new user ticks the box and picks Spain
- **THEN** they get a Spanish club and their account stores the acceptance time and the current policy version

### Requirement: Re-acceptance after a policy change
When the policy version is raised, every signed-in user whose accepted version is older SHALL be shown the updated terms and MUST accept them before using any other page, or may delete their account from that screen instead.

#### Scenario: Existing user after upgrade
- **WHEN** an existing user who never accepted any version signs in after this change ships
- **THEN** they see the accept-terms screen before reaching their dashboard

### Requirement: Only strictly necessary storage
The app SHALL use only storage needed for the service the user asked for: the HTTP-only session cookie, theme and language preferences, and the cookies Google's sign-in button sets. It MUST NOT add analytics, advertising or tracking without first adding a consent mechanism.

#### Scenario: First visit
- **WHEN** a visitor opens the home page for the first time
- **THEN** no request is made to any analytics or advertising service

### Requirement: Owner identity not exposed
Responses about clubs, players, matches and leagues SHALL NOT include any user's name, email or picture. The only exception for other players is a friend group: its members see each other's name, which the group page and the Privacy Policy disclose. Admin endpoints MAY show names and emails to admins.

#### Scenario: Viewing a rival club
- **WHEN** a user opens another human-owned club's page
- **THEN** the response contains no owner name or email

#### Scenario: Friend group
- **WHEN** a member opens their friend group's page
- **THEN** they see the other members' names and clubs, and never their emails

### Requirement: Legal identity from deployment properties
The controller identity shown in the legal pages SHALL come from deployment properties, not code: `LEGAL_CONTROLLER_NAME` and `LEGAL_CONTACT_EMAIL` (required), `LEGAL_CONTROLLER_TAX_ID`, `LEGAL_POSTAL_ADDRESS`, `LEGAL_HOSTING_PROVIDER` and `LEGAL_HOSTING_REGION` (optional). Changing a property MUST take effect on container restart without rebuilding.

#### Scenario: Properties set
- **WHEN** a deployment sets `LEGAL_CONTROLLER_NAME=Example Games SL` and `LEGAL_CONTACT_EMAIL=privacy@example.org`
- **THEN** the Privacy Policy names "Example Games SL" as controller and "privacy@example.org" as the contact for privacy requests

#### Scenario: Required property missing
- **WHEN** `LEGAL_CONTACT_EMAIL` is not set
- **THEN** the Privacy and Terms pages show a visible "legal details not configured" warning in place of the missing value

#### Scenario: Optional property missing
- **WHEN** `LEGAL_POSTAL_ADDRESS` is not set
- **THEN** the policy omits the address line instead of showing an empty value

### Requirement: Optional legal notice page
When the property `LEGAL_NOTICE_ENABLED` is `true`, the app SHALL serve a public legal notice ("Aviso legal") at `/legal-notice` in English and Spanish with the controller's name, tax id, postal address and contact email, linked wherever the Privacy Policy is linked. When it is not `true`, the page and its links MUST NOT appear.

#### Scenario: Enabled
- **WHEN** a deployment sets `LEGAL_NOTICE_ENABLED=true` with name, tax id and address
- **THEN** `/legal-notice` shows them and the login page links to it

#### Scenario: Disabled
- **WHEN** `LEGAL_NOTICE_ENABLED` is unset
- **THEN** no legal notice link is shown and `/legal-notice` falls through to the home page
