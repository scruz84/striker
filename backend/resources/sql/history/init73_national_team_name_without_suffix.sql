-- National teams stopped getting an " National Team" suffix baked into their stored name (see
-- common.nationalTeamName) - every list now localizes a national team's display name from its
-- country code on the frontend instead (COUNTRY_NAME_KEYS/localizedTeamName), so the stored name
-- itself is just the country's own name going forward. Existing rows created before this change
-- still carry the old suffix, so strip it here. Idempotent: once applied, no row still matches the
-- WHERE clause, so re-running this on every startup (see common.applyPendingMigrations) is a no-op.
UPDATE teams SET name = regexp_replace(name, ' National Team$', '')
WHERE is_national_team = true AND name LIKE '% National Team';
