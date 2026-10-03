-- Full baseline schema for a fresh database. This file used to be just the original tables,
-- with every later change shipped as its own resources/sql/initN_*.sql migration; those have
-- since been folded in here (most recently init37_lower_starting_budget.sql through
-- init58_stadium_capacity_default.sql) so a new database only needs to run this one file (plus the
-- names_*/lastnames_* name-pool data dumps, which stay separate as bulk data). The superseded
-- migration files still exist under resources/sql/history/ for reference - see
-- common.EnsureSchema for how the two directories are treated differently (only this directory's
-- *.sql files auto-apply).
--
-- Going forward, keep ARCHITECTURE.md's convention: new schema changes are additive
-- resources/sql/initN_*.sql migration files on top of this baseline, never edits to already-
-- shipped history and never edits to this file itself once it has shipped. This project isn't in
-- production yet, so an occasional flatten like this one (fold every current initN_*.sql into
-- this file, archive them to history/) is fine whenever it's convenient - once real deployments
-- exist, folding must stop being retroactive: existing databases would already have applied the
-- separate migration files via common.EnsureSchema's auto-apply, and re-running the same change
-- baked into init.sql would either no-op (idempotent DDL) or, for init.sql itself, error/duplicate
-- data, since it only ever runs against a genuinely empty database.

CREATE TABLE lastnames_es
(
     lastname varchar(100) NULL,
     freq float8 NULL
);

CREATE TABLE names_es
(
     "name" varchar(100) NULL,
     freq float8 NULL
);

CREATE TABLE countries (
    country_id 	bigserial NOT null,
    name varchar(4) null unique,
    PRIMARY KEY (country_id)
);

create table users (
   user_id 	bigserial NOT NULL,
   name 		varchar(200) not null,
   provider	varchar(100) not null,
   email		varchar(100) not null unique,
   country	varchar(4) references countries(name),
   is_admin boolean not null default false,
   -- Lets a user's chosen UI language (see frontend LanguageSwitcher.tsx) be known server-side
   -- too, so external notification delivery (email/push - see
   -- common.deliverExternalNotifications) can render in that language instead of always English.
   -- NULL (unset) means "no preference recorded yet" - falls back to English.
   language varchar(5),
   PRIMARY KEY (user_id)
);

CREATE TABLE teams (
   team_id 	bigserial NOT NULL,
   owner_id 	bigint ,
   country		varchar(4) references countries(name),
   league_id	bigint default -1,
   name varchar(200) NULL,
   -- Calibrated against common.PlayerWageFromSkills/common.SquadSize: an average-skill 18-player
   -- squad's wage bill alone runs ~28,350/week, so 400000 covers ~2 weeks shy of a full season
   -- (~18-20 match weeks) of wages even before any ticket/sponsorship/prize income - enough
   -- runway that a new manager isn't punished by early bad luck, but still low enough that squad
   -- and wage decisions matter well within the first season. The previous 1000000 covered ~35
   -- weeks (8+ months) of an average squad's wages on its own, before any income at all.
   budget bigint NOT NULL DEFAULT 400000,
   -- Every new team's starting stadium capacity - flat and fixed (see
   -- common.DefaultStadiumCapacity and the initializer's createTeam) rather than random or scaled
   -- by league tier, so every manager starts with the same-size ground. From here on it only ever
   -- changes through common.UpgradeStadium.
   stadium_capacity integer NOT NULL DEFAULT 1000,
   ticket_price integer NOT NULL DEFAULT 10,
   formation smallint NOT NULL DEFAULT 0,
   name_changed boolean NOT NULL DEFAULT false,
   ai_level smallint NOT NULL DEFAULT 0,
   -- Enforces the one-promotion-per-real-calendar-week cap on academy promotions (see
   -- common.CanPromoteFromAcademy) - a single timestamp is enough since at most one promotion can
   -- ever be "the most recent" for a team.
   last_academy_promotion_at timestamptz,
   -- A team's chosen sponsor deal (see common.SponsorDeal / engine.sponsorshipAmount) - a
   -- tradeoff between a higher guaranteed per-match sponsorship payment (0, Steady) and a lower
   -- guaranteed payment made up for by a bigger bonus on a win or draw (2, Performance), with 1
   -- (Balanced) in between. Defaults to Balanced so existing teams land on a reasonable middle
   -- ground rather than the flattest, lowest-upside option.
   sponsor_deal smallint NOT NULL DEFAULT 1,
   -- Locks a team's sponsor deal choice for the remainder of the season it was made in (issue
   -- #70) - previously a team could switch deals before every single match, defeating the point
   -- of a deal being a season-long tradeoff. 0 (no season is ever numbered 0 - see createSeason,
   -- which starts at 1) means "never explicitly chosen", so a team's very first choice is always
   -- allowed regardless of the current season number.
   sponsor_deal_season integer NOT NULL DEFAULT 0,
   -- A stadium's name (see common.RandomStadiumName) - null for a team created before this
   -- column existed, lazily backfilled by common.GetStadium the first time it's read.
   stadium_name varchar(200),
   -- A stadium capacity upgrade (see common.UpgradeStadium) takes common.StadiumUpgradeDuration
   -- to complete rather than applying instantly - stadium_upgrade_target is the capacity it will
   -- have once stadium_upgrade_completes_at passes; both null means no upgrade is in progress.
   stadium_upgrade_target integer,
   stadium_upgrade_completes_at timestamptz,
   PRIMARY KEY (team_id)
);

-- Singleton config row for the whole running world (there is always exactly one row).
-- match_schedule_weekdays is a bitmask, bit i set means time.Weekday(i) is a match day
-- (bit 0 = Sunday ... bit 6 = Saturday); default 127 selects every day.
-- match_schedule_minute_of_day is minutes since midnight (0-1439); default 960 is 16:00.
-- friendly_schedule_weekdays uses the same bitmask convention but has no time-of-day component
-- (see common.GetFriendlyScheduleWeekdays).
CREATE TABLE season (
    current_season int,
    transfer_window_open boolean NOT NULL DEFAULT true,
    match_schedule_weekdays smallint NOT NULL DEFAULT 127,
    match_schedule_minute_of_day smallint NOT NULL DEFAULT 960,
    max_friendlies_per_week smallint NOT NULL DEFAULT 1,
    friendly_schedule_weekdays smallint NOT NULL DEFAULT 127,
    -- Admin-configurable promotion/relegation slot counts (see
    -- common.GetPromotionRelegationConfig/SetPromotionRelegationConfig and
    -- engine.ComputePromotionRelegation). Defaults match the constants these replaced
    -- (engine.PromotionCandidatesPerGroup=1, RelegationSlotsPerGroup=2).
    promotion_slots_per_group smallint NOT NULL DEFAULT 1,
    relegation_slots_per_group smallint NOT NULL DEFAULT 2,
    -- Admin-configurable count of brand-new players injected straight into the free agent pool
    -- each real calendar week (see common.InjectFreeAgents/engine.InjectFreeAgentsJob). Zero
    -- turns injection off entirely.
    free_agent_injections_per_week smallint NOT NULL DEFAULT 5,
    -- Admin-configurable minimum time (in minutes) a received transfer offer must sit pending
    -- before it can be accepted (see common.GetOfferNegotiationWindowMinutes/common.AcceptOffer).
    -- Zero turns the window off entirely.
    offer_negotiation_window_minutes smallint NOT NULL DEFAULT 120
);

CREATE TABLE leagues (
     league_id 	bigserial NOT null,
     country	varchar(4) not null references countries(name),
     group_number	int not null,
     name 		varchar(10) not null,
     tier int NOT NULL DEFAULT 1,
     PRIMARY KEY (league_id)
);


-- A player's contract counts down by one every season (see engine.ProgressPlayers /
-- common.AdvancePlayerContract); when it hits zero they're released to free agency (team_id set
-- to NULL) rather than staying with their club forever. Any player with no team is signable by
-- any team via common.SignFreeAgent.
create table players (
     player_id         bigserial NOT null,
     team_id     bigint references teams(team_id),
     First_Name  varchar(200),
     Last_Name   varchar(200),
     Birth_Day   bigint,
     Goal_Keeper int,
     defense    int,
     "lateral"    int,
     Pass       int,
     Midfield   int,
     Dribbling  int,
     Shot       int,
     Header     bool,
     Quick      bool,
     Creative   bool,
     Technical  bool,
     avatar  varchar(1000),
     wage int NOT NULL DEFAULT 0,
     age int NOT NULL DEFAULT 22,
     contract_years_remaining int NOT NULL DEFAULT 0,
     -- The country whose name pool generated this player (see common.doCreatePlayers), independent
     -- of team_id's current country - the transfer market has no country restriction, so a player
     -- can end up playing anywhere regardless of where they were "born". Set once at creation,
     -- never touched again by a transfer, loan, or release.
     birth_country varchar(4) REFERENCES countries(name),
     -- Whether this player sits in the youth academy pool ("cantera") rather than the fieldable
     -- senior squad - see common.PromoteFromAcademy. A cantera prospect never counts toward
     -- common.SquadSize's automatic backfill floor, and only joins the senior squad, with a real
     -- wage and contract, once promoted.
     in_academy boolean NOT NULL DEFAULT false,
     -- How many consecutive seasons this player has gone unsigned (team_id IS NULL) - see
     -- engine.ProgressFreeAgents, the only writer, which increments this once per season and
     -- forces a free agent's retirement once it reaches engine.FreeAgencyRetirementSeasons. Reset
     -- to 0 the moment a player is signed (see common.SignFreeAgent).
     free_agent_seasons integer NOT NULL DEFAULT 0,
     -- Which club this free agent most recently played for, so signing them within their first
     -- season of free agency can charge a transfer premium to that club - see common.SignFreeAgent
     -- and common.FreeAgentPremiumWageMultiplier. NULL means either the player has never been
     -- released, or was unassigned by common.AssignTeam reassigning a CPU squad rather than a real
     -- release - neither case owes anyone a premium.
     last_team_id bigint REFERENCES teams(team_id),
     -- Squad number (issue #42): fixed for now (not user-editable), assigned by the application
     -- as the lowest number not already in use on a player's team (see
     -- common.nextFreeShirtNumber). A unique index enforcing one number per team is at the bottom
     -- of this file, alongside every other index. Academy prospects are left unnumbered until
     -- actually promoted (see common.PromoteFromAcademy).
     shirt_number integer,
     -- Player page-view counter, incremented every time PlayerDetailsHandler serves that
     -- player's profile - lets the transfer market stats panel surface the most-viewed listed
     -- player (issue #72).
     view_count bigint NOT NULL DEFAULT 0,
     PRIMARY KEY (player_id)
);

CREATE TABLE matches (
     match_id bigserial NOT null PRIMARY KEY,
     league_id int8 NOT NULL REFERENCES leagues(league_id),
     season int not null,
     played bool default false,
     in_simulation bool default false,
     week_number int default 0,
     match_date timestamptz NOT NULL,
     home_team_id int8 NOT NULL REFERENCES teams(team_id),
     away_team_id int8 NOT NULL REFERENCES teams(team_id),
     home_goals  int default 0,
     away_goals  int default 0,
     -- Match-report flavor (Hattrick-style match report): weather, referee/assistants, and the
     -- crowd size - generated once at simulation time (see common.UpdateMatchFlavor) so the
     -- report shows the same numbers/names on every later view rather than re-rolling them. NULL
     -- for a match that hasn't been simulated yet.
     weather varchar(20),
     attendance integer,
     referee_name varchar(120),
     assistant1_name varchar(120),
     assistant2_name varchar(120)
);

CREATE TABLE match_events (
    event_id bigserial not null primary key,
    match_id bigint not null references matches(match_id),
    event_type int,
    team_owner bigint references teams(team_id),
    other_team bigint references teams(team_id),
    minute int,
    team_players varchar(200),
    team_player_names varchar(1000),
    other_team_players varchar(200),
    other_team_player_names varchar(1000),
    home_goals int default 0,
    away_goals int default 0
);

CREATE TABLE match_instants (
      match_id bigint not null references matches(match_id),
      minute int not null,
      home_goals int default 0,
      away_goals int default 0,
      home_qua  varchar(500),
      away_qua  varchar(500),
      midfield_Diff         double precision,
      HomeAttack_Right_Diff  double precision,
      HomeAttack_Left_Diff   double precision,
      HomeAttack_Center_Diff double precision,
      AwayAttack_Right_Diff  double precision,
      AwayAttack_Left_Diff   double precision,
      AwayAttack_Center_Diff double precision,
      primary key (match_id, minute)
);


create table default_lineup_positions (
    team_id bigint references teams(team_id),
    player_id bigint references players(player_id) on delete set null,
    position smallint not null,
    primary key (team_id, player_id)
);

create table lineup (
    lineup_id bigserial primary key,
    match_id bigint references matches(match_id),
    team_id bigint references teams(team_id),
    -- Puts the previously-dormant lineup/lineup_positions tables to use as a per-league-match
    -- lineup override - common.GetLineupsForMatch already reads them with a fallback to
    -- default_lineup_positions; formation was the only piece missing to make this a complete
    -- mirror of friendly_lineup/friendly_lineup_positions.
    formation smallint NOT NULL DEFAULT 0,
    unique (match_id, team_id)
);

create table lineup_positions (
  lineup_id bigint references lineup(lineup_id),
  player_id bigint references players(player_id) on delete set null,
  position smallint not null
);

create table classification (
    league_id	bigint references leagues(league_id),
    team_id		bigint references teams(team_id),
    season  int not null,
    matches int default 0,
    points 	int default 0,
    wins 	int default 0,
    losses  int default 0,
    draws   int default 0,
    goals_scored int default 0,
    goals_received int default 0,
    primary key (league_id, team_id)
);

-- Tracks players currently unavailable through suspension (red card / second yellow) or injury,
-- and for how many more of their team's matches. See engine.applyUnavailability and
-- handlers.validateLineup.
CREATE TABLE player_unavailability (
    player_id        bigint PRIMARY KEY REFERENCES players(player_id) ON DELETE CASCADE,
    reason            smallint NOT NULL,
    matches_remaining int NOT NULL
);

-- Every time a player permanently joins a new club's senior squad - initial squad generation, a
-- transfer, a negotiated offer, or signing as a free agent (see common.RecordPlayerClubHistory
-- and its callers) - so a player's page can show the clubs they've played for, ordered by date.
-- Deliberately excludes loans (a temporary spell, not really "joining" a new club) and academy
-- promotion (the player was already on this same team's books, just in the cantera pool).
CREATE TABLE player_club_history (
    history_id bigserial PRIMARY KEY,
    player_id  bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    team_id    bigint NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
    joined_at  timestamptz NOT NULL DEFAULT now()
);

-- Team budgets, player wages, and a transaction ledger. See engine.deductWages and the transfer
-- market (handlers/transferHandlers.go) for what actually moves money. player_id ties a
-- transaction to the specific player it concerns (bought/sold, released early) - NULL for
-- team-level transactions (wages, ticket revenue, prize money, sponsorship, stadium upgrades,
-- admin adjustments) that aren't about any one player.
CREATE TABLE team_transactions (
    transaction_id bigserial PRIMARY KEY,
    team_id        bigint NOT NULL REFERENCES teams(team_id),
    amount         bigint NOT NULL, -- positive = income, negative = expense
    reason         smallint NOT NULL,
    description    varchar(500),
    created_at     timestamptz NOT NULL DEFAULT now(),
    player_id      bigint REFERENCES players(player_id) ON DELETE SET NULL
);

-- Transfer market listings. A listing is bought instantly at its asking price (no bidding/offer
-- negotiation) - see handlers/transferHandlers.go and common.ExecuteTransfer.
CREATE TABLE transfer_listings (
    listing_id      bigserial PRIMARY KEY,
    player_id       bigint NOT NULL UNIQUE REFERENCES players(player_id) ON DELETE CASCADE,
    selling_team_id bigint NOT NULL REFERENCES teams(team_id),
    asking_price    bigint NOT NULL,
    listed_at       timestamptz NOT NULL DEFAULT now()
);

-- A single-elimination cup competition per country per season, entrants drawn from the top
-- division (tier 1). Deliberately NOT built on the matches/match_events tables - those are
-- deeply wired to league classification (see engine.UpdateClassification), so a cup tie gets its
-- own tables instead. See engine/cups.go.
-- A cup can represent either a single country's own competition or an international one drawn
-- from several countries in the same region (see common.RegionForCountry) - exactly one of
-- country/region is set. cup_ties/cup_tie_events and everything downstream (bracket generation,
-- simulation, lineups) only ever reference team ids directly, with no country scoping, so no
-- other schema changes are needed for the international case.
CREATE TABLE cups (
    cup_id           bigserial PRIMARY KEY,
    country          varchar(4) REFERENCES countries(name),
    region           varchar(20),
    season           int NOT NULL,
    name             varchar(200) NOT NULL,
    completed        boolean NOT NULL DEFAULT false,
    champion_team_id bigint REFERENCES teams(team_id),
    CONSTRAINT cups_country_xor_region CHECK ((country IS NULL) <> (region IS NULL))
);

CREATE TABLE cup_ties (
    tie_id          bigserial PRIMARY KEY,
    cup_id          bigint NOT NULL REFERENCES cups(cup_id),
    round           int NOT NULL,
    slot_number     int NOT NULL,
    home_team_id    bigint REFERENCES teams(team_id),
    away_team_id    bigint REFERENCES teams(team_id),
    match_date      timestamptz,
    played          boolean NOT NULL DEFAULT false,
    in_simulation   boolean NOT NULL DEFAULT false,
    home_goals      int NOT NULL DEFAULT 0,
    away_goals      int NOT NULL DEFAULT 0,
    went_to_penalties boolean NOT NULL DEFAULT false,
    winner_team_id  bigint REFERENCES teams(team_id),
    UNIQUE (cup_id, round, slot_number)
);

CREATE TABLE cup_tie_events (
    event_id            bigserial PRIMARY KEY,
    tie_id              bigint NOT NULL REFERENCES cup_ties(tie_id),
    event_type          int,
    team_owner          bigint REFERENCES teams(team_id),
    other_team          bigint REFERENCES teams(team_id),
    minute              int,
    team_players        varchar(200),
    team_player_names   varchar(1000),
    home_goals          int DEFAULT 0,
    away_goals          int DEFAULT 0
);

-- Cup ties can NOT reuse lineup/lineup_positions the way league matches now do: cup_ties.tie_id
-- is its own independent bigserial sequence, unrelated to matches.match_id, so a tie and a
-- league match can easily share the same numeric id - engine.runMatchSimulation's own doc
-- comment already flagged this as unsafe the moment anything started writing match-keyed
-- override rows. Cup ties get their own tables instead, identically shaped, the same reasoning
-- friendly_lineup/friendly_lineup_positions already established for friendly_matches.match_id vs
-- matches.match_id.
CREATE TABLE cup_tie_lineup (
    lineup_id bigserial PRIMARY KEY,
    tie_id    bigint NOT NULL REFERENCES cup_ties(tie_id),
    team_id   bigint NOT NULL REFERENCES teams(team_id),
    formation smallint NOT NULL DEFAULT 0,
    UNIQUE (tie_id, team_id)
);

CREATE TABLE cup_tie_lineup_positions (
    lineup_id bigint NOT NULL REFERENCES cup_tie_lineup(lineup_id),
    player_id bigint REFERENCES players(player_id) ON DELETE SET NULL,
    position  smallint NOT NULL
);

-- Per-user notification log (match results, promotion/relegation, transfers, cup results). See
-- common/notifications.go for the generation points and engine wiring.
CREATE TABLE notifications (
    notification_id   bigserial PRIMARY KEY,
    user_id            bigint NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    notification_type  smallint NOT NULL,
    message            varchar(1000) NOT NULL,
    read               boolean NOT NULL DEFAULT false,
    created_at         timestamptz NOT NULL DEFAULT now(),
    -- Optional in-app route a notification click-throughs to (e.g. "/players/evolutions") - NULL
    -- for a notification type whose message alone says everything relevant.
    link               varchar(200),
    -- Lets a notification carry a frontend i18n key + params alongside its English message text,
    -- so the notification bell can render it in whichever language the viewer has chosen instead
    -- of always English (see common.NotificationMessage). Both nullable - a notification with
    -- neither falls back to the plain message text.
    message_key        varchar(64),
    message_params     jsonb
);

-- Player loans: a temporary, free transfer to another team that auto-returns the player to its
-- original owner at a set season. loan_listings is the open market (like transfer_listings);
-- accepting one creates a player_loans row tracking where the player has to go back to and when
-- - see common/loans.go and engine.ReturnExpiredLoans.
CREATE TABLE loan_listings (
    listing_id       bigserial PRIMARY KEY,
    player_id        bigint NOT NULL UNIQUE REFERENCES players(player_id) ON DELETE CASCADE,
    lending_team_id  bigint NOT NULL REFERENCES teams(team_id),
    duration_seasons int NOT NULL,
    listed_at        timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE player_loans (
    loan_id       bigserial PRIMARY KEY,
    player_id     bigint NOT NULL UNIQUE REFERENCES players(player_id) ON DELETE CASCADE,
    from_team_id  bigint NOT NULL REFERENCES teams(team_id),
    to_team_id    bigint NOT NULL REFERENCES teams(team_id),
    return_season int NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- Direct transfer offers ("scouting"): unlike transfer_listings (a seller advertises a fixed
-- price), here a buyer proposes an amount for any player, listed or not, and the current owner
-- accepts or rejects. Every row is a pending offer - accepted/rejected/withdrawn ones are
-- deleted rather than archived, same as transfer_listings/loan_listings. See common/offers.go.
CREATE TABLE transfer_offers (
    offer_id      bigserial PRIMARY KEY,
    player_id     bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    from_team_id  bigint NOT NULL REFERENCES teams(team_id),
    to_team_id    bigint NOT NULL REFERENCES teams(team_id),
    amount        bigint NOT NULL,
    -- The personal terms (wage/contract length) the buyer is proposing to the player, alongside
    -- the fee proposed to the selling club - see common.MakeTransferOffer/AcceptOffer.
    offered_wage  integer NOT NULL DEFAULT 0,
    offered_years integer NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (player_id, from_team_id)
);

-- Append-only log of transfer market activity - negotiated offers (see common.MakeTransferOffer)
-- and outright buys of a listed player (see common.ExecuteTransfer), from human and AI-managed
-- teams alike - unlike transfer_offers itself, which is deleted the moment an offer resolves
-- (accepted/rejected/withdrawn), this never shrinks, so "how many offers were made in the last
-- hour/day" (issue #72) stays answerable regardless of how quickly offers get resolved.
CREATE TABLE transfer_offer_log (
    event_id   bigserial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- A team's trophy cabinet: one row per (team, season), independent of the current-season
-- classification table (which is reset in place at each rollover, so it can never hold last
-- season's standings). Written from two independent places that can each fire first depending
-- on timing, so every column but the key is nullable and both writers upsert: engine's
-- recordSeasonHistory (league final position + promotion/relegation, at season rollover) and
-- recordCupChampion (cup_result, whenever a cup completes).
CREATE TABLE season_history (
    team_id        bigint NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
    season         int NOT NULL,
    league_id      bigint,
    league_name    varchar(200),
    final_position int,
    promoted       boolean NOT NULL DEFAULT false,
    relegated      boolean NOT NULL DEFAULT false,
    cup_result     varchar(50),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, season)
);

-- Friend groups: a lightweight social leaderboard among teams the group's members pick from the
-- existing shared world - NOT isolated private leagues. See common/groups.go.
CREATE TABLE groups (
    group_id    bigserial PRIMARY KEY,
    name        varchar(100) NOT NULL,
    owner_id    bigint NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    invite_code varchar(20) NOT NULL UNIQUE,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE group_members (
    group_id  bigint NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    user_id   bigint NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    joined_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (group_id, user_id)
);

CREATE TABLE notification_preferences (
    user_id       bigint NOT NULL PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    email_enabled boolean NOT NULL DEFAULT false,
    push_enabled  boolean NOT NULL DEFAULT false,
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- A browser's Web Push subscription (see PushManager.subscribe on the frontend) for one of a
-- user's devices/browsers - a user can have several (phone, laptop, ...), each getting its own
-- row, so a notification goes out to every device they've enabled push on. endpoint is unique
-- per subscription (the browser's own push service URL), so re-subscribing the same
-- device/browser just replaces its keys rather than accumulating duplicates. Deleted once its
-- endpoint starts reporting expired/unsubscribed (a 404/410 from the push service - see
-- common.SendWebPush) or the user disables push in their notification preferences.
CREATE TABLE push_subscriptions (
    subscription_id bigserial PRIMARY KEY,
    user_id         bigint NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    endpoint        varchar(500) NOT NULL UNIQUE,
    p256dh          varchar(255) NOT NULL,
    auth            varchar(255) NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);

-- Name pools for more seeded countries (en/fr/de/it/pt), mirroring the names_es/lastnames_es
-- schema. Curated common first/last names (~60 each) rather than a dump the size of
-- names_es/lastnames_es (~29k rows) - a deliberately smaller pool, disclosed as a scope cut: more
-- repetition across a large squad of teams than the Spanish pool has, but real, common names.
-- it/pt are NOT inserted into `countries`/SeededCountries here - they stay inactive until an
-- admin explicitly activates one via POST /v1/admin/countries (see common.KnownCountries).
CREATE TABLE names_en ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_en (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_fr ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_fr (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_de ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_de (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_it ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_it (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_pt ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_pt (lastname varchar(100) NULL, freq float8 NULL);

INSERT INTO names_en ("name", freq) VALUES
('James',1.0),('John',1.0),('Robert',1.0),('Michael',1.0),('William',1.0),('David',1.0),('Richard',1.0),
('Joseph',1.0),('Thomas',1.0),('Charles',1.0),('Christopher',1.0),('Daniel',1.0),('Matthew',1.0),
('Anthony',1.0),('Mark',1.0),('Paul',1.0),('Steven',1.0),('Andrew',1.0),('Kenneth',1.0),('George',1.0),
('Edward',1.0),('Brian',1.0),('Ronald',1.0),('Timothy',1.0),('Jason',1.0),('Jeffrey',1.0),('Gary',1.0),
('Ryan',1.0),('Nicholas',1.0),('Eric',1.0),('Stephen',1.0),('Jacob',1.0),('Larry',1.0),('Jonathan',1.0),
('Frank',1.0),('Scott',1.0),('Justin',1.0),('Brandon',1.0),('Raymond',1.0),('Gregory',1.0),('Samuel',1.0),
('Benjamin',1.0),('Patrick',1.0),('Jack',1.0),('Dennis',1.0),('Jerry',1.0),('Alexander',1.0),('Tyler',1.0),
('Henry',1.0),('Douglas',1.0),('Peter',1.0),('Walter',1.0),('Harold',1.0),('Adam',1.0),('Arthur',1.0),
('Nathan',1.0),('Wayne',1.0),('Zachary',1.0),('Kyle',1.0),('Carl',1.0);

INSERT INTO lastnames_en (lastname, freq) VALUES
('Smith',1.0),('Jones',1.0),('Taylor',1.0),('Williams',1.0),('Brown',1.0),('Davies',1.0),('Evans',1.0),
('Wilson',1.0),('Thomas',1.0),('Roberts',1.0),('Johnson',1.0),('Lewis',1.0),('Walker',1.0),('Robinson',1.0),
('Wood',1.0),('Thompson',1.0),('White',1.0),('Watson',1.0),('Jackson',1.0),('Wright',1.0),('Green',1.0),
('Harris',1.0),('Cooper',1.0),('King',1.0),('Lee',1.0),('Baker',1.0),('Clarke',1.0),('James',1.0),
('Hall',1.0),('Turner',1.0),('Parker',1.0),('Edwards',1.0),('Hughes',1.0),('Hill',1.0),('Moore',1.0),
('Clark',1.0),('Bennett',1.0),('Morris',1.0),('Miller',1.0),('Ward',1.0),('Cook',1.0),('Rogers',1.0),
('Morgan',1.0),('Bell',1.0),('Cox',1.0),('Richardson',1.0),('Bailey',1.0),('Collins',1.0),('Bird',1.0),
('Foster',1.0),('Fisher',1.0),('Gray',1.0),('Hunt',1.0),('Knight',1.0),('Palmer',1.0),('Reid',1.0),
('Shaw',1.0),('Stevens',1.0),('Webb',1.0),('Young',1.0);

INSERT INTO names_fr ("name", freq) VALUES
('Jean',1.0),('Pierre',1.0),('Michel',1.0),('Andre',1.0),('Philippe',1.0),('Alain',1.0),('Jacques',1.0),
('Bernard',1.0),('Daniel',1.0),('Christian',1.0),('Claude',1.0),('Robert',1.0),('Nicolas',1.0),('Rene',1.0),
('Louis',1.0),('Marcel',1.0),('Georges',1.0),('Paul',1.0),('Francois',1.0),('Henri',1.0),('Roger',1.0),
('Guy',1.0),('Gerard',1.0),('Serge',1.0),('Marc',1.0),('Patrick',1.0),('Denis',1.0),('Yves',1.0),
('Antoine',1.0),('Julien',1.0),('Thomas',1.0),('Vincent',1.0),('Olivier',1.0),('Christophe',1.0),('David',1.0),
('Laurent',1.0),('Stephane',1.0),('Eric',1.0),('Frederic',1.0),('Sebastien',1.0),('Alexandre',1.0),('Maxime',1.0),
('Romain',1.0),('Baptiste',1.0),('Gabriel',1.0),('Hugo',1.0),('Lucas',1.0),('Mathieu',1.0),('Guillaume',1.0),
('Simon',1.0),('Theo',1.0),('Leo',1.0),('Adrien',1.0),('Benjamin',1.0),('Damien',1.0),('Fabrice',1.0),
('Jerome',1.0),('Ludovic',1.0),('Pascal',1.0),('Thierry',1.0),('Xavier',1.0);

INSERT INTO lastnames_fr (lastname, freq) VALUES
('Martin',1.0),('Bernard',1.0),('Thomas',1.0),('Petit',1.0),('Robert',1.0),('Richard',1.0),('Durand',1.0),
('Dubois',1.0),('Moreau',1.0),('Laurent',1.0),('Simon',1.0),('Michel',1.0),('Lefebvre',1.0),('Leroy',1.0),
('Roux',1.0),('David',1.0),('Bertrand',1.0),('Morel',1.0),('Fournier',1.0),('Girard',1.0),('Bonnet',1.0),
('Dupont',1.0),('Lambert',1.0),('Fontaine',1.0),('Rousseau',1.0),('Vincent',1.0),('Muller',1.0),('Lefevre',1.0),
('Faure',1.0),('Andre',1.0),('Mercier',1.0),('Blanc',1.0),('Guerin',1.0),('Boyer',1.0),('Garnier',1.0),
('Chevalier',1.0),('Francois',1.0),('Legrand',1.0),('Gauthier',1.0),('Garcia',1.0),('Perrin',1.0),('Robin',1.0),
('Clement',1.0),('Morin',1.0),('Nicolas',1.0),('Henry',1.0),('Roussel',1.0),('Mathieu',1.0),('Gautier',1.0),
('Masson',1.0),('Marchand',1.0),('Duval',1.0),('Denis',1.0),('Dumont',1.0),('Marie',1.0),('Lemoine',1.0),
('Meyer',1.0),('Noel',1.0),('Meunier',1.0),('Rolland',1.0);

INSERT INTO names_de ("name", freq) VALUES
('Hans',1.0),('Karl',1.0),('Klaus',1.0),('Werner',1.0),('Wolfgang',1.0),('Manfred',1.0),('Peter',1.0),
('Michael',1.0),('Thomas',1.0),('Andreas',1.0),('Juergen',1.0),('Helmut',1.0),('Guenther',1.0),('Dieter',1.0),
('Rainer',1.0),('Uwe',1.0),('Stefan',1.0),('Frank',1.0),('Bernd',1.0),('Norbert',1.0),('Heinz',1.0),
('Rolf',1.0),('Horst',1.0),('Walter',1.0),('Erich',1.0),('Kurt',1.0),('Gerhard',1.0),('Herbert',1.0),
('Otto',1.0),('Wilhelm',1.0),('Christian',1.0),('Markus',1.0),('Ralf',1.0),('Matthias',1.0),('Joerg',1.0),
('Christoph',1.0),('Sebastian',1.0),('Florian',1.0),('Daniel',1.0),('Alexander',1.0),('Tobias',1.0),
('Martin',1.0),('Benjamin',1.0),('Maximilian',1.0),('Felix',1.0),('Jonas',1.0),('Lukas',1.0),('Julian',1.0),
('Fabian',1.0),('Simon',1.0),('Philipp',1.0),('Jan',1.0),('David',1.0),('Niklas',1.0),('Leon',1.0),
('Paul',1.0),('Erik',1.0),('Marcel',1.0),('Dennis',1.0),('Sven',1.0);

INSERT INTO lastnames_de (lastname, freq) VALUES
('Mueller',1.0),('Schmidt',1.0),('Schneider',1.0),('Fischer',1.0),('Weber',1.0),('Meyer',1.0),('Wagner',1.0),
('Becker',1.0),('Schulz',1.0),('Hoffmann',1.0),('Schaefer',1.0),('Koch',1.0),('Bauer',1.0),('Richter',1.0),
('Klein',1.0),('Wolf',1.0),('Schroeder',1.0),('Neumann',1.0),('Schwarz',1.0),('Zimmermann',1.0),('Braun',1.0),
('Krueger',1.0),('Hofmann',1.0),('Hartmann',1.0),('Lange',1.0),('Schmitt',1.0),('Werner',1.0),('Krause',1.0),
('Meier',1.0),('Lehmann',1.0),('Schmid',1.0),('Schulze',1.0),('Maier',1.0),('Koehler',1.0),('Herrmann',1.0),
('Koenig',1.0),('Walter',1.0),('Mayer',1.0),('Huber',1.0),('Kaiser',1.0),('Fuchs',1.0),('Peters',1.0),
('Lang',1.0),('Scholz',1.0),('Moeller',1.0),('Weiss',1.0),('Jung',1.0),('Hahn',1.0),('Schubert',1.0),
('Vogel',1.0),('Friedrich',1.0),('Keller',1.0),('Guenther',1.0),('Frank',1.0),('Berger',1.0),('Winkler',1.0),
('Roth',1.0),('Beck',1.0),('Lorenz',1.0),('Baumann',1.0);

INSERT INTO names_it ("name", freq) VALUES
('Giuseppe',1.0),('Giovanni',1.0),('Antonio',1.0),('Mario',1.0),('Luigi',1.0),('Francesco',1.0),
('Angelo',1.0),('Vincenzo',1.0),('Pietro',1.0),('Salvatore',1.0),('Franco',1.0),('Domenico',1.0),
('Bruno',1.0),('Carlo',1.0),('Alberto',1.0),('Sergio',1.0),('Roberto',1.0),('Marco',1.0),
('Paolo',1.0),('Renato',1.0),('Aldo',1.0),('Alfredo',1.0),('Ugo',1.0),('Dario',1.0),
('Massimo',1.0),('Fabio',1.0),('Claudio',1.0),('Enrico',1.0),('Riccardo',1.0),('Stefano',1.0),
('Luca',1.0),('Andrea',1.0),('Matteo',1.0),('Davide',1.0),('Simone',1.0),('Alessandro',1.0),
('Federico',1.0),('Nicola',1.0),('Gabriele',1.0),('Emanuele',1.0),('Lorenzo',1.0),('Tommaso',1.0),
('Giacomo',1.0),('Leonardo',1.0),('Cristiano',1.0),('Filippo',1.0),('Michele',1.0),('Gianni',1.0),
('Raffaele',1.0),('Ettore',1.0),('Ivano',1.0),('Sandro',1.0),('Gino',1.0),('Cesare',1.0),
('Emilio',1.0),('Ferdinando',1.0),('Amedeo',1.0),('Guido',1.0),('Silvio',1.0),('Nicolo',1.0);

INSERT INTO lastnames_it (lastname, freq) VALUES
('Rossi',1.0),('Russo',1.0),('Ferrari',1.0),('Esposito',1.0),('Bianchi',1.0),('Romano',1.0),
('Colombo',1.0),('Ricci',1.0),('Marino',1.0),('Greco',1.0),('Bruno',1.0),('Gallo',1.0),
('Conti',1.0),('Mancini',1.0),('Costa',1.0),('Giordano',1.0),('Rizzo',1.0),('Lombardi',1.0),
('Moretti',1.0),('Barbieri',1.0),('Fontana',1.0),('Santoro',1.0),('Mariani',1.0),('Rinaldi',1.0),
('Caruso',1.0),('Ferrara',1.0),('Galli',1.0),('Martini',1.0),('Leone',1.0),('Longo',1.0),
('Gentile',1.0),('Martinelli',1.0),('Vitale',1.0),('Lombardo',1.0),('Serra',1.0),('Coppola',1.0),
('Marchetti',1.0),('Parisi',1.0),('Villa',1.0),('Conte',1.0),('Ferraro',1.0),('Ferri',1.0),
('Fabbri',1.0),('Bianco',1.0),('Marini',1.0),('Grasso',1.0),('Valentini',1.0),('Messina',1.0),
('Sala',1.0),('Gatti',1.0),('Pellegrini',1.0),('Palumbo',1.0),('Sanna',1.0),('Farina',1.0),
('Rizzi',1.0),('Monti',1.0),('Testa',1.0),('Silvestri',1.0),('Bellini',1.0),('Fiore',1.0);

INSERT INTO names_pt ("name", freq) VALUES
('Joao',1.0),('Jose',1.0),('Antonio',1.0),('Manuel',1.0),('Francisco',1.0),('Carlos',1.0),
('Paulo',1.0),('Pedro',1.0),('Luis',1.0),('Miguel',1.0),('Rui',1.0),('Bruno',1.0),
('Nuno',1.0),('Ricardo',1.0),('Fernando',1.0),('Jorge',1.0),('Tiago',1.0),('Andre',1.0),
('Diogo',1.0),('Hugo',1.0),('Marco',1.0),('Vitor',1.0),('Sergio',1.0),('Filipe',1.0),
('Rodrigo',1.0),('Gustavo',1.0),('Alexandre',1.0),('Duarte',1.0),('Goncalo',1.0),('Simao',1.0),
('Afonso',1.0),('Martim',1.0),('Tomas',1.0),('Guilherme',1.0),('Henrique',1.0),('Leonardo',1.0),
('Daniel',1.0),('Gabriel',1.0),('Rafael',1.0),('Vasco',1.0),('Mario',1.0),('Alberto',1.0),
('Artur',1.0),('Bernardo',1.0),('Cristiano',1.0),('Diego',1.0),('Eduardo',1.0),('Emanuel',1.0),
('Frederico',1.0),('Ivo',1.0),('Joel',1.0),('Leandro',1.0),('Marcelo',1.0),('Renato',1.0),
('Sandro',1.0),('Valter',1.0),('Xavier',1.0),('Amadeu',1.0),('Osvaldo',1.0),('Anibal',1.0);

INSERT INTO lastnames_pt (lastname, freq) VALUES
('Silva',1.0),('Santos',1.0),('Ferreira',1.0),('Pereira',1.0),('Oliveira',1.0),('Costa',1.0),
('Rodrigues',1.0),('Martins',1.0),('Sousa',1.0),('Fernandes',1.0),('Goncalves',1.0),('Gomes',1.0),
('Lopes',1.0),('Marques',1.0),('Alves',1.0),('Almeida',1.0),('Ribeiro',1.0),('Pinto',1.0),
('Carvalho',1.0),('Teixeira',1.0),('Moreira',1.0),('Correia',1.0),('Mendes',1.0),('Nunes',1.0),
('Soares',1.0),('Vieira',1.0),('Monteiro',1.0),('Cardoso',1.0),('Rocha',1.0),('Neves',1.0),
('Coelho',1.0),('Cruz',1.0),('Cunha',1.0),('Pires',1.0),('Ramos',1.0),('Reis',1.0),
('Simoes',1.0),('Antunes',1.0),('Matos',1.0),('Fonseca',1.0),('Machado',1.0),('Araujo',1.0),
('Barbosa',1.0),('Tavares',1.0),('Pacheco',1.0),('Nogueira',1.0),('Freitas',1.0),('Amaral',1.0),
('Batista',1.0),('Guerreiro',1.0),('Pinheiro',1.0),('Melo',1.0),('Faria',1.0),('Duarte',1.0),
('Loureiro',1.0),('Xavier',1.0),('Salgado',1.0),('Pimentel',1.0),('Peixoto',1.0),('Braga',1.0);

-- Adds the game's first non-European countries (issue #63): Brazil, Japan, and Nigeria, one from
-- each of the three continents the issue asked for beyond Europe. Each needs its own
-- names_<code>/lastnames_<code> name-pool pair (see common.GetRandomName/nameTableSuffixByCountry)
-- - team/stadium naming data lives in Go (see common/teamnames.go, common/stadiumnames.go), not
-- the database. Countries stay inactive (not in the `countries` table) until an admin activates
-- one via POST /v1/admin/countries, same as it/pt.

CREATE TABLE names_br ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_br (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_jp ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_jp (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_ng ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_ng (lastname varchar(100) NULL, freq float8 NULL);

INSERT INTO names_br ("name", freq) VALUES
    ('Joao',1.0),('Jose',1.0),('Pedro',1.0),('Paulo',1.0),('Carlos',1.0),('Lucas',1.0),('Gabriel',1.0),
    ('Rafael',1.0),('Bruno',1.0),('Gustavo',1.0),('Rodrigo',1.0),('Felipe',1.0),('Thiago',1.0),('Marcelo',1.0),
    ('Ricardo',1.0),('Fernando',1.0),('Eduardo',1.0),('Diego',1.0),('Vinicius',1.0),('Matheus',1.0),('Leonardo',1.0),
    ('Daniel',1.0),('Marcos',1.0),('Anderson',1.0),('Alexandre',1.0),('Fabio',1.0),('Renato',1.0),('Wesley',1.0),
    ('Everton',1.0),('Cristiano',1.0),('Douglas',1.0),('Robson',1.0),('Wagner',1.0),('Adriano',1.0),('Jefferson',1.0),
    ('Elias',1.0),('Emerson',1.0),('Junior',1.0),('Cassio',1.0),('Igor',1.0),('Vitor',1.0),('Caio',1.0),
    ('Henrique',1.0),('Guilherme',1.0),('Andre',1.0),('Luan',1.0),('Kaio',1.0),('Renan',1.0),('Willian',1.0),
    ('Ronaldo',1.0),('Roberto',1.0),('Marcio',1.0);

INSERT INTO lastnames_br (lastname, freq) VALUES
    ('Silva',1.0),('Santos',1.0),('Oliveira',1.0),('Souza',1.0),('Pereira',1.0),('Costa',1.0),('Rodrigues',1.0),
    ('Almeida',1.0),('Nascimento',1.0),('Lima',1.0),('Araujo',1.0),('Fernandes',1.0),('Carvalho',1.0),('Gomes',1.0),
    ('Martins',1.0),('Rocha',1.0),('Ribeiro',1.0),('Alves',1.0),('Monteiro',1.0),('Mendes',1.0),('Barros',1.0),
    ('Freitas',1.0),('Barbosa',1.0),('Pinto',1.0),('Moreira',1.0),('Cardoso',1.0),('Correia',1.0),('Teixeira',1.0),
    ('Cavalcanti',1.0),('Machado',1.0),('Ramos',1.0),('Vieira',1.0),('Dias',1.0),('Batista',1.0),('Nunes',1.0),
    ('Marques',1.0),('Castro',1.0),('Andrade',1.0),('Campos',1.0),('Cunha',1.0),('Melo',1.0),('Farias',1.0),
    ('Moura',1.0),('Azevedo',1.0),('Reis',1.0),('Sales',1.0),('Xavier',1.0),('Tavares',1.0),('Coelho',1.0),
    ('Brito',1.0),('Duarte',1.0),('Guimaraes',1.0),('Neves',1.0);

INSERT INTO names_jp ("name", freq) VALUES
    ('Haruto',1.0),('Yuto',1.0),('Sota',1.0),('Ryo',1.0),('Kaito',1.0),('Riku',1.0),('Sora',1.0),('Yuki',1.0),
    ('Hayato',1.0),('Taiga',1.0),('Kota',1.0),('Shota',1.0),('Kenta',1.0),('Daiki',1.0),('Kazuki',1.0),('Shun',1.0),
    ('Ren',1.0),('Tsubasa',1.0),('Naoki',1.0),('Takumi',1.0),('Yuma',1.0),('Kohei',1.0),('Ryota',1.0),('Satoshi',1.0),
    ('Yusuke',1.0),('Kenji',1.0),('Hiroshi',1.0),('Takashi',1.0),('Masato',1.0),('Akira',1.0),('Wataru',1.0),
    ('Tomoya',1.0),('Keisuke',1.0),('Genki',1.0),('Shinji',1.0),('Junya',1.0),('Hidetoshi',1.0),('Atsuto',1.0),
    ('Makoto',1.0),('Shoya',1.0),('Ritsu',1.0),('Sho',1.0),('Rikuto',1.0),('Asahi',1.0),('Minato',1.0),('Itsuki',1.0),
    ('Haru',1.0),('Sosuke',1.0),('Yamato',1.0),('Rento',1.0),('Aoi',1.0);

INSERT INTO lastnames_jp (lastname, freq) VALUES
    ('Sato',1.0),('Suzuki',1.0),('Takahashi',1.0),('Tanaka',1.0),('Watanabe',1.0),('Ito',1.0),('Yamamoto',1.0),
    ('Nakamura',1.0),('Kobayashi',1.0),('Kato',1.0),('Yoshida',1.0),('Yamada',1.0),('Sasaki',1.0),('Yamaguchi',1.0),
    ('Matsumoto',1.0),('Inoue',1.0),('Kimura',1.0),('Hayashi',1.0),('Shimizu',1.0),('Yamazaki',1.0),('Mori',1.0),
    ('Abe',1.0),('Ikeda',1.0),('Hashimoto',1.0),('Yamashita',1.0),('Ishikawa',1.0),('Nakajima',1.0),('Maeda',1.0),
    ('Fujita',1.0),('Ogawa',1.0),('Goto',1.0),('Okada',1.0),('Hasegawa',1.0),('Murakami',1.0),('Kondo',1.0),
    ('Ishii',1.0),('Saito',1.0),('Sakamoto',1.0),('Endo',1.0),('Aoki',1.0),('Fujii',1.0),('Nishimura',1.0),
    ('Fukuda',1.0),('Ota',1.0),('Miura',1.0),('Fujiwara',1.0),('Okamoto',1.0),('Matsuda',1.0),('Nakagawa',1.0),
    ('Nakano',1.0),('Harada',1.0),('Ono',1.0);

INSERT INTO names_ng ("name", freq) VALUES
    ('Chukwuemeka',1.0),('Emmanuel',1.0),('Ifeanyi',1.0),('Chidi',1.0),('Obinna',1.0),('Uche',1.0),('Chinedu',1.0),
    ('Ikechukwu',1.0),('Emeka',1.0),('Kelechi',1.0),('Chibuzor',1.0),('Victor',1.0),('Samuel',1.0),('Daniel',1.0),
    ('Joseph',1.0),('John',1.0),('Peter',1.0),('Paul',1.0),('Michael',1.0),('David',1.0),('Success',1.0),
    ('Godwin',1.0),('Sunday',1.0),('Friday',1.0),('Blessing',1.0),('Kingsley',1.0),('Ahmed',1.0),('Musa',1.0),
    ('Ibrahim',1.0),('Abdullahi',1.0),('Yusuf',1.0),('Aliyu',1.0),('Suleiman',1.0),('Umar',1.0),('Adamu',1.0),
    ('Bello',1.0),('Tunde',1.0),('Segun',1.0),('Femi',1.0),('Bayo',1.0),('Wale',1.0),('Yemi',1.0),('Kunle',1.0),
    ('Dele',1.0),('Gbenga',1.0),('Kayode',1.0),('Olumide',1.0),('Adewale',1.0),('Babatunde',1.0),('Adeyemi',1.0),
    ('Efe',1.0);

INSERT INTO lastnames_ng (lastname, freq) VALUES
    ('Okafor',1.0),('Okonkwo',1.0),('Eze',1.0),('Nwosu',1.0),('Nwachukwu',1.0),('Ibe',1.0),('Chukwu',1.0),('Obi',1.0),
    ('Anyanwu',1.0),('Uzoma',1.0),('Adebayo',1.0),('Ogundipe',1.0),('Balogun',1.0),('Afolabi',1.0),('Bakare',1.0),
    ('Ojo',1.0),('Ogunleye',1.0),('Fashola',1.0),('Okoro',1.0),('Oladipo',1.0),('Ojukwu',1.0),('Danjuma',1.0),
    ('Abubakar',1.0),('Mohammed',1.0),('Usman',1.0),('Garba',1.0),('Sani',1.0),('Yakubu',1.0),('Lawal',1.0),
    ('Shehu',1.0),('Ndubuisi',1.0),('Onyeka',1.0),('Ikenna',1.0),('Nnamdi',1.0),('Okoli',1.0),('Nnaji',1.0),
    ('Anozie',1.0),('Ugwu',1.0),('Onyekwere',1.0),('Madueke',1.0),('Iwobi',1.0),('Ndidi',1.0),('Etebo',1.0),
    ('Chukwueze',1.0),('Aina',1.0),('Ekong',1.0),('Uzoho',1.0),('Osimhen',1.0),('Simon',1.0),('Iheanacho',1.0);

-- Rounds every region (see common.RegionForCountry) out to at least ten countries: 4 more for
-- Europe, 9 more for the Americas, 9 more for Asia, 9 more for Africa. Same pattern as the
-- br/jp/ng pools above: each country needs its own names_<code>/lastnames_<code> pair (see
-- common.GetRandomName/nameTableSuffixByCountry) - team/stadium naming data lives in Go (see
-- common/teamnames.go, common/stadiumnames.go), not the database. Countries stay inactive (not
-- in the `countries` table) until an admin activates one via POST /v1/admin/countries.

CREATE TABLE names_nl ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_nl (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_se ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_se (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_pl ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_pl (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_gr ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_gr (lastname varchar(100) NULL, freq float8 NULL);

CREATE TABLE names_ar ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_ar (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_mx ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_mx (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_co ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_co (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_cl ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_cl (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_pe ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_pe (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_uy ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_uy (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_us ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_us (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_ca ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_ca (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_ec ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_ec (lastname varchar(100) NULL, freq float8 NULL);

CREATE TABLE names_kr ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_kr (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_cn ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_cn (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_in ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_in (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_th ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_th (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_vn ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_vn (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_id ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_id (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_ph ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_ph (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_sa ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_sa (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_ir ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_ir (lastname varchar(100) NULL, freq float8 NULL);

CREATE TABLE names_eg ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_eg (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_gh ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_gh (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_ci ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_ci (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_cm ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_cm (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_sn ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_sn (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_za ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_za (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_ma ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_ma (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_dz ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_dz (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE names_ke ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE lastnames_ke (lastname varchar(100) NULL, freq float8 NULL);

-- Europe: Netherlands, Sweden, Poland, Greece

INSERT INTO names_nl ("name", freq) VALUES
    ('Daan',1.0),('Sem',1.0),('Lucas',1.0),('Milan',1.0),('Levi',1.0),('Luuk',1.0),('Bram',1.0),
    ('Jesse',1.0),('Finn',1.0),('Noah',1.0),('Thijs',1.0),('Sven',1.0),('Tim',1.0),('Ruben',1.0),
    ('Niels',1.0),('Joris',1.0),('Tom',1.0),('Max',1.0),('Stijn',1.0),('Cas',1.0),('Twan',1.0),
    ('Bart',1.0),('Wouter',1.0),('Jelle',1.0),('Koen',1.0),('Mees',1.0),('Pim',1.0),('Job',1.0),
    ('Dylan',1.0),('Kai',1.0),('Guus',1.0),('Boaz',1.0),('Sander',1.0),('Rick',1.0),('Wesley',1.0),
    ('Dennis',1.0),('Marco',1.0),('Erik',1.0),('Bas',1.0),('Robin',1.0);

INSERT INTO lastnames_nl (lastname, freq) VALUES
    ('de Jong',1.0),('de Vries',1.0),('Jansen',1.0),('van den Berg',1.0),('van Dijk',1.0),('Bakker',1.0),
    ('Visser',1.0),('Smit',1.0),('Meijer',1.0),('Boer',1.0),('Mulder',1.0),('de Groot',1.0),
    ('Bos',1.0),('Vos',1.0),('Peters',1.0),('Hendriks',1.0),('Dekker',1.0),('Brouwer',1.0),
    ('van Leeuwen',1.0),('Dijkstra',1.0),('Smits',1.0),('de Wit',1.0),('Hoekstra',1.0),('van der Meer',1.0),
    ('de Boer',1.0),('Kok',1.0),('Postma',1.0),('Vermeulen',1.0),('van der Linden',1.0),('Bosman',1.0),
    ('van der Heijden',1.0),('Willems',1.0),('Kuiper',1.0),('van der Wal',1.0),('van Beek',1.0),
    ('van Vliet',1.0),('Verhoeven',1.0),('Verheul',1.0),('van Loon',1.0),('Timmermans',1.0);

INSERT INTO names_se ("name", freq) VALUES
    ('Erik',1.0),('Lars',1.0),('Anders',1.0),('Johan',1.0),('Karl',1.0),('Nils',1.0),('Per',1.0),
    ('Gustav',1.0),('Oskar',1.0),('Emil',1.0),('Axel',1.0),('Filip',1.0),('Viktor',1.0),('Anton',1.0),
    ('Simon',1.0),('Hugo',1.0),('Elias',1.0),('Oliver',1.0),('William',1.0),('Alexander',1.0),
    ('Isak',1.0),('Leo',1.0),('Adam',1.0),('Fredrik',1.0),('Magnus',1.0),('Daniel',1.0),('Marcus',1.0),
    ('Henrik',1.0),('Mattias',1.0),('Jonas',1.0),('Tobias',1.0),('Rasmus',1.0),('Sebastian',1.0),
    ('Linus',1.0),('Martin',1.0),('Jesper',1.0),('Robin',1.0),('Andreas',1.0),('Kristian',1.0),('Niklas',1.0);

INSERT INTO lastnames_se (lastname, freq) VALUES
    ('Andersson',1.0),('Johansson',1.0),('Karlsson',1.0),('Nilsson',1.0),('Eriksson',1.0),('Larsson',1.0),
    ('Olsson',1.0),('Persson',1.0),('Svensson',1.0),('Gustafsson',1.0),('Pettersson',1.0),('Jonsson',1.0),
    ('Jansson',1.0),('Hansson',1.0),('Bengtsson',1.0),('Jonsson',1.0),('Lindberg',1.0),('Jakobsson',1.0),
    ('Magnusson',1.0),('Olofsson',1.0),('Lindqvist',1.0),('Lindgren',1.0),('Berg',1.0),('Bergstrom',1.0),
    ('Lundberg',1.0),('Lundqvist',1.0),('Sandberg',1.0),('Forsberg',1.0),('Nystrom',1.0),('Holmberg',1.0),
    ('Wallin',1.0),('Engstrom',1.0),('Sjoberg',1.0),('Lindstrom',1.0),('Aberg',1.0),('Ekstrom',1.0),
    ('Blom',1.0),('Nordin',1.0),('Bjork',1.0),('Soderberg',1.0);

INSERT INTO names_pl ("name", freq) VALUES
    ('Jakub',1.0),('Kacper',1.0),('Antoni',1.0),('Filip',1.0),('Jan',1.0),('Szymon',1.0),('Franciszek',1.0),
    ('Aleksander',1.0),('Wojciech',1.0),('Mateusz',1.0),('Michal',1.0),('Bartosz',1.0),('Piotr',1.0),
    ('Tomasz',1.0),('Krzysztof',1.0),('Marcin',1.0),('Adam',1.0),('Pawel',1.0),('Lukasz',1.0),('Rafal',1.0),
    ('Dawid',1.0),('Damian',1.0),('Patryk',1.0),('Kamil',1.0),('Sebastian',1.0),('Grzegorz',1.0),
    ('Marek',1.0),('Andrzej',1.0),('Stanislaw',1.0),('Zbigniew',1.0),('Jerzy',1.0),('Ryszard',1.0),
    ('Wladyslaw',1.0),('Henryk',1.0),('Tadeusz',1.0),('Mariusz',1.0),('Robert',1.0),('Artur',1.0),
    ('Norbert',1.0),('Ignacy',1.0);

INSERT INTO lastnames_pl (lastname, freq) VALUES
    ('Nowak',1.0),('Kowalski',1.0),('Wisniewski',1.0),('Wojcik',1.0),('Kowalczyk',1.0),('Kaminski',1.0),
    ('Lewandowski',1.0),('Zielinski',1.0),('Szymanski',1.0),('Wozniak',1.0),('Dabrowski',1.0),('Kozlowski',1.0),
    ('Jankowski',1.0),('Mazur',1.0),('Wojciechowski',1.0),('Kwiatkowski',1.0),('Krawczyk',1.0),('Piotrowski',1.0),
    ('Grabowski',1.0),('Zajac',1.0),('Krol',1.0),('Wieczorek',1.0),('Jablonski',1.0),('Wrobel',1.0),
    ('Nowakowski',1.0),('Majewski',1.0),('Olszewski',1.0),('Stepien',1.0),('Malinowski',1.0),('Jaworski',1.0),
    ('Adamczyk',1.0),('Dudek',1.0),('Nowicki',1.0),('Pawlowski',1.0),('Michalski',1.0),('Krzeminski',1.0),
    ('Zalewski',1.0),('Sikora',1.0),('Walczak',1.0),('Baran',1.0);

INSERT INTO names_gr ("name", freq) VALUES
    ('Georgios',1.0),('Ioannis',1.0),('Konstantinos',1.0),('Dimitrios',1.0),('Nikolaos',1.0),('Panagiotis',1.0),
    ('Vasilios',1.0),('Christos',1.0),('Michail',1.0),('Athanasios',1.0),('Andreas',1.0),('Stavros',1.0),
    ('Ilias',1.0),('Apostolos',1.0),('Grigorios',1.0),('Spyridon',1.0),('Theodoros',1.0),('Emmanouil',1.0),
    ('Alexandros',1.0),('Petros',1.0),('Charalampos',1.0),('Aristeidis',1.0),('Kyriakos',1.0),('Sotirios',1.0),
    ('Anestis',1.0),('Efstathios',1.0),('Dionysios',1.0),('Angelos',1.0),('Fotios',1.0),('Leonidas',1.0),
    ('Marios',1.0),('Nestor',1.0),('Orestis',1.0),('Panos',1.0),('Rafail',1.0),('Savvas',1.0),
    ('Timoleon',1.0),('Vlasis',1.0),('Yannis',1.0),('Zisis',1.0);

INSERT INTO lastnames_gr (lastname, freq) VALUES
    ('Papadopoulos',1.0),('Papadimitriou',1.0),('Papageorgiou',1.0),('Ioannou',1.0),('Georgiou',1.0),
    ('Nikolaidis',1.0),('Dimitriou',1.0),('Vasiliou',1.0),('Konstantinou',1.0),('Christodoulou',1.0),
    ('Antoniou',1.0),('Athanasiou',1.0),('Papanikolaou',1.0),('Karagiannis',1.0),('Economou',1.0),
    ('Angelopoulos',1.0),('Makris',1.0),('Stavrou',1.0),('Pappas',1.0),('Katsaros',1.0),
    ('Triantafyllou',1.0),('Xenakis',1.0),('Stamatiou',1.0),('Panagiotopoulos',1.0),('Sideris',1.0),
    ('Kyriakou',1.0),('Michailidis',1.0),('Petrou',1.0),('Sotiriou',1.0),('Fotiadis',1.0),
    ('Grigoriou',1.0),('Andreou',1.0),('Zisimopoulos',1.0),('Vlachos',1.0),('Karydis',1.0),
    ('Mitsis',1.0),('Voulgaris',1.0),('Kontos',1.0),('Lazaridis',1.0),('Moraitis',1.0);

-- Americas: Argentina, Mexico, Colombia, Chile, Peru, Uruguay, United States, Canada, Ecuador

INSERT INTO names_ar ("name", freq) VALUES
    ('Lionel',1.0),('Sergio',1.0),('Angel',1.0),('Nicolas',1.0),('Rodrigo',1.0),('Emiliano',1.0),('Lautaro',1.0),
    ('Julian',1.0),('Alejandro',1.0),('Marcos',1.0),('Facundo',1.0),('Franco',1.0),('Nahuel',1.0),
    ('Exequiel',1.0),('Enzo',1.0),('Thiago',1.0),('Santiago',1.0),('Mateo',1.0),('Bruno',1.0),('Ivan',1.0),
    ('Ezequiel',1.0),('Cristian',1.0),('German',1.0),('Leandro',1.0),('Maximiliano',1.0),('Gonzalo',1.0),
    ('Federico',1.0),('Agustin',1.0),('Joaquin',1.0),('Tomas',1.0),('Ramiro',1.0),('Ignacio',1.0),
    ('Guido',1.0),('Valentin',1.0),('Dario',1.0),('Hernan',1.0),('Matias',1.0),('Damian',1.0),
    ('Walter',1.0),('Osvaldo',1.0);

INSERT INTO lastnames_ar (lastname, freq) VALUES
    ('Gonzalez',1.0),('Rodriguez',1.0),('Fernandez',1.0),('Lopez',1.0),('Martinez',1.0),('Perez',1.0),
    ('Garcia',1.0),('Sanchez',1.0),('Romero',1.0),('Diaz',1.0),('Alvarez',1.0),('Torres',1.0),
    ('Ruiz',1.0),('Ramirez',1.0),('Flores',1.0),('Acosta',1.0),('Benitez',1.0),('Medina',1.0),
    ('Herrera',1.0),('Suarez',1.0),('Aguirre',1.0),('Molina',1.0),('Ortiz',1.0),('Silva',1.0),
    ('Rojas',1.0),('Nunez',1.0),('Gimenez',1.0),('Castro',1.0),('Vega',1.0),('Rios',1.0),
    ('Cabrera',1.0),('Ferreyra',1.0),('Godoy',1.0),('Ledesma',1.0),('Paez',1.0),('Quiroga',1.0),
    ('Sosa',1.0),('Villalba',1.0),('Zarate',1.0),('Bustos',1.0);

INSERT INTO names_mx ("name", freq) VALUES
    ('Jose',1.0),('Juan',1.0),('Carlos',1.0),('Luis',1.0),('Miguel',1.0),('Jesus',1.0),('Alejandro',1.0),
    ('Francisco',1.0),('Ricardo',1.0),('Fernando',1.0),('Roberto',1.0),('Eduardo',1.0),('Manuel',1.0),
    ('Antonio',1.0),('Sergio',1.0),('Raul',1.0),('Victor',1.0),('Alberto',1.0),('Arturo',1.0),('Ramon',1.0),
    ('Hugo',1.0),('Ivan',1.0),('Oscar',1.0),('Cesar',1.0),('Jorge',1.0),('Guillermo',1.0),('Rafael',1.0),
    ('Marco',1.0),('Enrique',1.0),('Gerardo',1.0),('Salvador',1.0),('Emilio',1.0),('Rodolfo',1.0),
    ('Adrian',1.0),('Israel',1.0),('Ulises',1.0),('Gustavo',1.0),('Erick',1.0),('Alfonso',1.0),('Rene',1.0);

INSERT INTO lastnames_mx (lastname, freq) VALUES
    ('Hernandez',1.0),('Garcia',1.0),('Martinez',1.0),('Lopez',1.0),('Gonzalez',1.0),('Perez',1.0),
    ('Rodriguez',1.0),('Sanchez',1.0),('Ramirez',1.0),('Flores',1.0),('Torres',1.0),('Diaz',1.0),
    ('Vazquez',1.0),('Reyes',1.0),('Cruz',1.0),('Morales',1.0),('Ortiz',1.0),('Gutierrez',1.0),
    ('Chavez',1.0),('Ramos',1.0),('Mendoza',1.0),('Ruiz',1.0),('Aguilar',1.0),('Jimenez',1.0),
    ('Moreno',1.0),('Romero',1.0),('Alvarez',1.0),('Medina',1.0),('Castillo',1.0),('Vargas',1.0),
    ('Guzman',1.0),('Salazar',1.0),('Contreras',1.0),('Rios',1.0),('Delgado',1.0),('Fuentes',1.0),
    ('Cervantes',1.0),('Marin',1.0),('Cortez',1.0),('Espinoza',1.0);

INSERT INTO names_co ("name", freq) VALUES
    ('Andres',1.0),('Carlos',1.0),('Camilo',1.0),('Santiago',1.0),('Sebastian',1.0),('Juan',1.0),('Cristian',1.0),
    ('Jhon',1.0),('Yerry',1.0),('Wilmar',1.0),('Duvan',1.0),('Miguel',1.0),('Radamel',1.0),('Luis',1.0),
    ('Jefferson',1.0),('Mateus',1.0),('Rafael',1.0),('Wilson',1.0),('Yeison',1.0),('Jorge',1.0),
    ('Fabian',1.0),('Alexis',1.0),('Cesar',1.0),('Jhonier',1.0),('Dayro',1.0),('Edwin',1.0),('Harold',1.0),
    ('Frank',1.0),('Alvaro',1.0),('Gustavo',1.0),('Nelson',1.0),('Diego',1.0),('Elkin',1.0),('Farid',1.0),
    ('Gerardo',1.0),('Kevin',1.0),('Marlon',1.0),('Steven',1.0),('Yimmi',1.0);

INSERT INTO lastnames_co (lastname, freq) VALUES
    ('Rodriguez',1.0),('Gomez',1.0),('Martinez',1.0),('Lopez',1.0),('Garcia',1.0),('Perez',1.0),
    ('Gonzalez',1.0),('Sanchez',1.0),('Ramirez',1.0),('Torres',1.0),('Diaz',1.0),('Vargas',1.0),
    ('Castro',1.0),('Ruiz',1.0),('Alvarez',1.0),('Romero',1.0),('Suarez',1.0),('Rojas',1.0),
    ('Moreno',1.0),('Ortiz',1.0),('Munoz',1.0),('Herrera',1.0),('Jimenez',1.0),('Ramos',1.0),
    ('Reyes',1.0),('Cardona',1.0),('Zapata',1.0),('Mejia',1.0),('Palacios',1.0),('Valencia',1.0),
    ('Arias',1.0),('Marin',1.0),('Cadavid',1.0),('Cuadrado',1.0),('Uribe',1.0),('Muriel',1.0),
    ('Guerra',1.0),('Osorio',1.0),('Pardo',1.0);

INSERT INTO names_cl ("name", freq) VALUES
    ('Alexis',1.0),('Arturo',1.0),('Gary',1.0),('Claudio',1.0),('Charles',1.0),('Eduardo',1.0),('Mauricio',1.0),
    ('Gonzalo',1.0),('Marcelo',1.0),('Jean',1.0),('Erick',1.0),('Ben',1.0),('Diego',1.0),('Nicolas',1.0),
    ('Bruno',1.0),('Guillermo',1.0),('Fabian',1.0),('Angelo',1.0),('Sebastian',1.0),('Junior',1.0),
    ('Igor',1.0),('Ivan',1.0),('Esteban',1.0),('Felipe',1.0),('Cristopher',1.0),('Rodrigo',1.0),
    ('Enzo',1.0),('Matias',1.0),('Pablo',1.0),('Ignacio',1.0),('Vicente',1.0),('Tomas',1.0),
    ('Cristobal',1.0),('Joaquin',1.0),('Maximiliano',1.0),('Benjamin',1.0),('Agustin',1.0),('Martin',1.0),
    ('Ricardo',1.0),('Hernan',1.0);

INSERT INTO lastnames_cl (lastname, freq) VALUES
    ('Sanchez',1.0),('Sanhueza',1.0),('Aranguiz',1.0),('Valdes',1.0),('Vidal',1.0),('Isla',1.0),
    ('Medel',1.0),('Pinilla',1.0),('Diaz',1.0),('Vargas',1.0),('Silva',1.0),('Munoz',1.0),
    ('Rojas',1.0),('Fuentes',1.0),('Reyes',1.0),('Torres',1.0),('Espinoza',1.0),('Contreras',1.0),
    ('Araya',1.0),('Tapia',1.0),('Soto',1.0),('Riquelme',1.0),('Cortes',1.0),('Vera',1.0),
    ('Sepulveda',1.0),('Bravo',1.0),('Carrasco',1.0),('Fernandez',1.0),('Gutierrez',1.0),('Molina',1.0),
    ('Pizarro',1.0),('Zuniga',1.0),('Toro',1.0),('Salinas',1.0),('Orellana',1.0),('Guzman',1.0),
    ('Herrera',1.0),('Ortiz',1.0),('Rivera',1.0),('Escobar',1.0);

INSERT INTO names_pe ("name", freq) VALUES
    ('Paolo',1.0),('Christian',1.0),('Jefferson',1.0),('Andre',1.0),('Yoshimar',1.0),('Edison',1.0),('Renato',1.0),
    ('Miguel',1.0),('Pedro',1.0),('Carlos',1.0),('Luis',1.0),('Gianluca',1.0),('Alexander',1.0),('Wilder',1.0),
    ('Anderson',1.0),('Aldair',1.0),('Sergio',1.0),('Raul',1.0),('Alex',1.0),('Cristian',1.0),
    ('Ray',1.0),('Beto',1.0),('Marcos',1.0),('Jose',1.0),('Franco',1.0),('Ricardo',1.0),('Alberto',1.0),
    ('Jean',1.0),('Kevin',1.0),('Nilson',1.0),('Percy',1.0),('Alexis',1.0),('Diego',1.0),('Mauricio',1.0),
    ('Cesar',1.0),('Christofer',1.0),('Renzo',1.0),('Piero',1.0),('Junior',1.0),('Fernando',1.0);

INSERT INTO lastnames_pe (lastname, freq) VALUES
    ('Guerrero',1.0),('Farfan',1.0),('Vargas',1.0),('Cueva',1.0),('Flores',1.0),('Advincula',1.0),
    ('Trauco',1.0),('Zambrano',1.0),('Carrillo',1.0),('Yotun',1.0),('Tapia',1.0),('Lapadula',1.0),
    ('Ramos',1.0),('Lopez',1.0),('Rodriguez',1.0),('Chavez',1.0),('Rojas',1.0),('Castillo',1.0),
    ('Reyes',1.0),('Torres',1.0),('Salazar',1.0),('Quispe',1.0),('Mamani',1.0),('Huaman',1.0),
    ('Condori',1.0),('Aguilar',1.0),('Cardenas',1.0),('Mendoza',1.0),('Palomino',1.0),('Zevallos',1.0),
    ('Ponce',1.0),('Delgado',1.0),('Villanueva',1.0),('Nunez',1.0),('Cruzado',1.0),('Guevara',1.0),
    ('Solano',1.0),('Requena',1.0);

INSERT INTO names_uy ("name", freq) VALUES
    ('Luis',1.0),('Edinson',1.0),('Diego',1.0),('Federico',1.0),('Jose',1.0),('Nahitan',1.0),('Rodrigo',1.0),
    ('Matias',1.0),('Fede',1.0),('Ronald',1.0),('Martin',1.0),('Giorgian',1.0),('Lucas',1.0),('Sebastian',1.0),
    ('Gaston',1.0),('Cristian',1.0),('Bruno',1.0),('Nicolas',1.0),('Maximiliano',1.0),('Guillermo',1.0),
    ('Gonzalo',1.0),('Facundo',1.0),('Agustin',1.0),('Sergio',1.0),('Alvaro',1.0),('Mathias',1.0),
    ('Fabian',1.0),('Jonathan',1.0),('Emiliano',1.0),('Franco',1.0),('Damian',1.0),('Ignacio',1.0),
    ('Marcelo',1.0),('Walter',1.0),('Andres',1.0),('Pablo',1.0),('Fernando',1.0),('Leandro',1.0);

INSERT INTO lastnames_uy (lastname, freq) VALUES
    ('Suarez',1.0),('Cavani',1.0),('Godin',1.0),('Bentancur',1.0),('Valverde',1.0),('Araujo',1.0),
    ('Torreira',1.0),('Vecino',1.0),('Nunez',1.0),('Rodriguez',1.0),('Pereira',1.0),('Coates',1.0),
    ('Gimenez',1.0),('Cristoforo',1.0),('Rolan',1.0),('Cabrera',1.0),('Vina',1.0),('Olivera',1.0),
    ('Pellistri',1.0),('Ugarte',1.0),('Ramirez',1.0),('Silva',1.0),('Fernandez',1.0),('Gonzalez',1.0),
    ('Perez',1.0),('Lopez',1.0),('Martinez',1.0),('Alonso',1.0),('Sosa',1.0),('Machado',1.0),
    ('Rivas',1.0),('Techera',1.0),('Corujo',1.0),('Guzman',1.0),('Cardozo',1.0),('Britos',1.0),
    ('Amaral',1.0);

INSERT INTO names_us ("name", freq) VALUES
    ('Christian',1.0),('Weston',1.0),('Tyler',1.0),('Gio',1.0),('Sergino',1.0),('Tim',1.0),('Brenden',1.0),
    ('Ricardo',1.0),('Josh',1.0),('Antonee',1.0),('Walker',1.0),('Matt',1.0),('Zack',1.0),('Yunus',1.0),
    ('Paul',1.0),('Landon',1.0),('Clint',1.0),('Michael',1.0),('Jozy',1.0),('Kellyn',1.0),
    ('Cole',1.0),('Aaron',1.0),('Brandon',1.0),('Cameron',1.0),('Erik',1.0),('Jordan',1.0),('Reggie',1.0),
    ('Jesse',1.0),('John',1.0),('Kyle',1.0),('Nick',1.0),('Ryan',1.0),('Scott',1.0),('Steve',1.0),
    ('Brian',1.0),('Frankie',1.0),('Sam',1.0),('Ethan',1.0),('Justin',1.0);

INSERT INTO lastnames_us (lastname, freq) VALUES
    ('Pulisic',1.0),('McKennie',1.0),('Reyna',1.0),('Adams',1.0),('Dest',1.0),('Ream',1.0),
    ('Aaronson',1.0),('Turner',1.0),('Robinson',1.0),('Sargent',1.0),('Musah',1.0),('Zimmerman',1.0),
    ('Scally',1.0),('Weah',1.0),('Yedlin',1.0),('Bradley',1.0),('Howard',1.0),('Dempsey',1.0),
    ('Donovan',1.0),('Altidore',1.0),('Wood',1.0),('Zardes',1.0),('Morris',1.0),('Acosta',1.0),
    ('Roldan',1.0),('Arriola',1.0),('Boyd',1.0),('Lletget',1.0),('Long',1.0),('Miazga',1.0),
    ('Ledezma',1.0),('Pepi',1.0),('Balogun',1.0),('Tillman',1.0),('Richards',1.0),('Horvath',1.0),
    ('Steffen',1.0),('Vazquez',1.0);

INSERT INTO names_ca ("name", freq) VALUES
    ('Alphonso',1.0),('Jonathan',1.0),('Cyle',1.0),('Tajon',1.0),('Stephen',1.0),('Junior',1.0),('Sam',1.0),
    ('Jamal',1.0),('Alistair',1.0),('Milan',1.0),('Kamal',1.0),('Liam',1.0),('Doneil',1.0),('Jayden',1.0),
    ('Richie',1.0),('David',1.0),('Atiba',1.0),('Scott',1.0),('Ike',1.0),('Dejan',1.0),
    ('Maxime',1.0),('Ismael',1.0),('Lucas',1.0),('Samuel',1.0),('Jacen',1.0),('Kadin',1.0),('Zachary',1.0),
    ('Marco',1.0),('Steven',1.0),('Russell',1.0),('Dominick',1.0),('Marcus',1.0),('Derek',1.0),
    ('Adam',1.0),('Terry',1.0),('Dean',1.0),('Julian',1.0),('Charles',1.0);

INSERT INTO lastnames_ca (lastname, freq) VALUES
    ('Davies',1.0),('David',1.0),('Larin',1.0),('Buchanan',1.0),('Eustaquio',1.0),('Osorio',1.0),
    ('Adekugbe',1.0),('Hutchinson',1.0),('Johnston',1.0),('Kone',1.0),('Miller',1.0),('Cornelius',1.0),
    ('Millar',1.0),('Laryea',1.0),('Cavallini',1.0),('Piette',1.0),('Wotherspoon',1.0),('Henry',1.0),
    ('Hoilett',1.0),('Bustos',1.0),('Waterman',1.0),('James',1.0),('Vitoria',1.0),
    ('Akindele',1.0),('Ledgerwood',1.0),('Borjan',1.0),('Crepeau',1.0),('Doyle',1.0),
    ('Priso',1.0),('Roberts',1.0),('Petrasso',1.0),('Kaye',1.0),('Shaffelburg',1.0),
    ('Brault-Guillard',1.0),('Jackson',1.0),('Baldisimo',1.0),('Metcalfe',1.0);

INSERT INTO names_ec ("name", freq) VALUES
    ('Enner',1.0),('Michael',1.0),('Moises',1.0),('Byron',1.0),('Pervis',1.0),('Piero',1.0),('Angelo',1.0),
    ('Gonzalo',1.0),('Alexander',1.0),('Felix',1.0),('Jefferson',1.0),('Robert',1.0),('Xavier',1.0),
    ('Antonio',1.0),('Jhegson',1.0),('Renato',1.0),('Jose',1.0),('Fernando',1.0),('Cristian',1.0),
    ('Pedro',1.0),('Diego',1.0),('Carlos',1.0),('Miguel',1.0),('Alan',1.0),('Adrian',1.0),('Nilson',1.0),
    ('Bryan',1.0),('Kevin',1.0),('Ronny',1.0),('Cristhian',1.0),('Rodrigo',1.0),('Damian',1.0),
    ('Israel',1.0),('Jean',1.0),('Ricardo',1.0),('Junior',1.0),('Marlon',1.0),('Oscar',1.0),('Wellington',1.0);

INSERT INTO lastnames_ec (lastname, freq) VALUES
    ('Valencia',1.0),('Caicedo',1.0),('Preciado',1.0),('Estrada',1.0),('Mendez',1.0),('Ibarra',1.0),
    ('Plata',1.0),('Hincapie',1.0),('Pacho',1.0),('Sarmiento',1.0),('Torres',1.0),('Arroyo',1.0),
    ('Cifuentes',1.0),('Mena',1.0),('Angulo',1.0),('Quinonez',1.0),('Rodriguez',1.0),('Perlaza',1.0),
    ('Gruezo',1.0),('Franco',1.0),('Bone',1.0),('Cazares',1.0),('Quinteros',1.0),('Chala',1.0),
    ('Castillo',1.0),('Vera',1.0),('Zambrano',1.0),('Delgado',1.0),('Macias',1.0),('Segura',1.0),
    ('Lopez',1.0),('Tenorio',1.0),('Reasco',1.0),('Cortez',1.0),('Minda',1.0),('Congo',1.0),
    ('Ordonez',1.0);

-- Asia: South Korea, China, India, Thailand, Vietnam, Indonesia, Philippines, Saudi Arabia, Iran

INSERT INTO names_kr ("name", freq) VALUES
    ('Min-jae',1.0),('Heung-min',1.0),('Kang-in',1.0),('Woo-yeong',1.0),('Hee-chan',1.0),('Jae-sung',1.0),
    ('In-beom',1.0),('Seung-ho',1.0),('Young-gwon',1.0),('Dong-gyeong',1.0),('Hyun-woo',1.0),('Kyung-won',1.0),
    ('Jung-hoon',1.0),('Chang-hoon',1.0),('Do-hyun',1.0),('Ui-jo',1.0),('Jong-woo',1.0),('Jin-su',1.0),
    ('Jae-hyeok',1.0),('Sung-yueng',1.0),('Seon-min',1.0),('Chan-ho',1.0),('Tae-hwan',1.0),
    ('Chung-yong',1.0),('Hyung-jin',1.0),('Jung-woo',1.0),('Yun-do',1.0),('Sung-hoon',1.0),('Ye-chan',1.0),
    ('Woo-hyun',1.0),('Kyu-sung',1.0),('Se-jong',1.0),('Do-yeon',1.0),('Jae-won',1.0),('Seong-yoon',1.0),
    ('Han-bin',1.0),('Yong-woo',1.0);

INSERT INTO lastnames_kr (lastname, freq) VALUES
    ('Kim',1.0),('Lee',1.0),('Park',1.0),('Choi',1.0),('Jung',1.0),('Kang',1.0),('Cho',1.0),
    ('Yoon',1.0),('Jang',1.0),('Lim',1.0),('Han',1.0),('Oh',1.0),('Seo',1.0),('Shin',1.0),
    ('Kwon',1.0),('Hwang',1.0),('Ahn',1.0),('Song',1.0),('Yoo',1.0),('Hong',1.0),
    ('Ko',1.0),('Moon',1.0),('Yang',1.0),('Bae',1.0),('Baek',1.0),('Nam',1.0),('Ryu',1.0),
    ('Ha',1.0),('Gu',1.0),('Ok',1.0),('Chun',1.0),('Ju',1.0),('Do',1.0),('Byun',1.0),
    ('Jeon',1.0),('Gong',1.0),('Rho',1.0),('Sim',1.0),('Woo',1.0);

INSERT INTO names_cn ("name", freq) VALUES
    ('Wei',1.0),('Jun',1.0),('Hao',1.0),('Yang',1.0),('Lei',1.0),('Peng',1.0),('Bo',1.0),('Chao',1.0),
    ('Fei',1.0),('Gang',1.0),('Tao',1.0),('Kai',1.0),('Cheng',1.0),('Hui',1.0),('Jian',1.0),('Long',1.0),
    ('Ming',1.0),('Qiang',1.0),('Wen',1.0),('Xin',1.0),('Yong',1.0),('Zhi',1.0),('Ping',1.0),('Feng',1.0),
    ('Hong',1.0),('Guo',1.0),('Bin',1.0),('Da',1.0),('Rui',1.0),('Xiang',1.0),('Zheng',1.0),('Jia',1.0),
    ('Song',1.0),('Liang',1.0),('Bao',1.0),('Fan',1.0),('Han',1.0),('Xun',1.0),('Zhen',1.0);

INSERT INTO lastnames_cn (lastname, freq) VALUES
    ('Wang',1.0),('Li',1.0),('Zhang',1.0),('Liu',1.0),('Chen',1.0),('Yang',1.0),('Huang',1.0),('Zhao',1.0),
    ('Wu',1.0),('Zhou',1.0),('Xu',1.0),('Sun',1.0),('Ma',1.0),('Zhu',1.0),('Hu',1.0),('Guo',1.0),
    ('He',1.0),('Lin',1.0),('Gao',1.0),('Luo',1.0),('Zheng',1.0),('Liang',1.0),('Xie',1.0),('Song',1.0),
    ('Tang',1.0),('Han',1.0),('Feng',1.0),('Deng',1.0),('Cao',1.0),('Peng',1.0),('Zeng',1.0),('Xiao',1.0),
    ('Tian',1.0),('Dong',1.0),('Yuan',1.0),('Pan',1.0),('Cai',1.0),('Jiang',1.0),('Yu',1.0),('Du',1.0);

INSERT INTO names_in ("name", freq) VALUES
    ('Sunil',1.0),('Sandesh',1.0),('Gurpreet',1.0),('Anirudh',1.0),('Rahul',1.0),('Sahal',1.0),('Manvir',1.0),
    ('Udanta',1.0),('Liston',1.0),('Pritam',1.0),('Adil',1.0),('Amarjit',1.0),('Anwar',1.0),('Ashique',1.0),
    ('Brandon',1.0),('Deepak',1.0),('Jeakson',1.0),('Nikhil',1.0),
    ('Pronay',1.0),('Rowllin',1.0),('Suresh',1.0),('Vishal',1.0),('Yumnam',1.0),('Akash',1.0),('Bikash',1.0),
    ('Farukh',1.0),('Gurmeet',1.0),('Harmanjot',1.0),('Ishan',1.0),('Jitendra',1.0),('Karan',1.0),
    ('Milan',1.0),('Naorem',1.0),('Omkar',1.0),('Prabir',1.0),('Rajat',1.0),('Simranjeet',1.0);

INSERT INTO lastnames_in (lastname, freq) VALUES
    ('Chhetri',1.0),('Singh',1.0),('Sandhu',1.0),('Kumar',1.0),('Yadav',1.0),('Sharma',1.0),('Verma',1.0),
    ('Gupta',1.0),('Mehta',1.0),('Kapoor',1.0),('Chauhan',1.0),('Rana',1.0),('Thapa',1.0),('Gowda',1.0),
    ('Nair',1.0),('Menon',1.0),('Iyer',1.0),('Reddy',1.0),('Rao',1.0),('Pillai',1.0),
    ('Bora',1.0),('Khan',1.0),('Ali',1.0),('Ansari',1.0),('Shaikh',1.0),('Chowdhury',1.0),('Bose',1.0),
    ('Dutta',1.0),('Sengupta',1.0),('Mukherjee',1.0),('Banerjee',1.0),('Ghosh',1.0),('Das',1.0),
    ('Roy',1.0),('Sarkar',1.0),('Bhatt',1.0),('Joshi',1.0),('Malhotra',1.0);

INSERT INTO names_th ("name", freq) VALUES
    ('Chanathip',1.0),('Theerathon',1.0),('Teerasil',1.0),('Adisak',1.0),('Sarach',1.0),('Kawin',1.0),
    ('Pansa',1.0),('Supachok',1.0),('Manuel',1.0),('Elias',1.0),('Bordin',1.0),('Peeradol',1.0),
    ('Nitipong',1.0),('Sasalak',1.0),('Weerathep',1.0),('Thitipan',1.0),('Korrakot',1.0),('Ekanit',1.0),
    ('Siwakorn',1.0),('Anon',1.0),('Ben',1.0),('Chananan',1.0),('Ekkachai',1.0),
    ('Firdaus',1.0),('Godpop',1.0),('Issawa',1.0),('Jetsada',1.0),('Kritsada',1.0),
    ('Mongkol',1.0),('Narit',1.0),('Peerapat',1.0),('Rangsan',1.0),('Sittichok',1.0),('Tanaboon',1.0),
    ('Watcharapong',1.0);

INSERT INTO lastnames_th (lastname, freq) VALUES
    ('Songkrasin',1.0),('Chaikwan',1.0),('Nualsri',1.0),('Yoonan',1.0),('Dangda',1.0),('Pokklaw',1.0),
    ('Chotnamniyom',1.0),('Anusorn',1.0),('Panyachalerm',1.0),('Suthirat',1.0),('Aiyarawit',1.0),
    ('Kesorn',1.0),('Preedee',1.0),('Sompong',1.0),('Rattanakorn',1.0),('Thawornsak',1.0),
    ('Wongsuwan',1.0),('Wichaya',1.0),('Sakulwiwatana',1.0),('Nualpakdee',1.0),
    ('Boonlorm',1.0),('Rattanawan',1.0),('Thongchai',1.0),('Sirisophon',1.0),('Panpim',1.0),
    ('Kittikun',1.0),('Sombat',1.0),('Chartpanit',1.0),('Wongchan',1.0),('Suksawat',1.0),
    ('Boonprasert',1.0),('Thanasan',1.0),('Manirot',1.0),('Pattamakiat',1.0);

INSERT INTO names_vn ("name", freq) VALUES
    ('Van',1.0),('Cong',1.0),('Quang',1.0),('Duc',1.0),('Minh',1.0),('Hoang',1.0),('Anh',1.0),('Tuan',1.0),
    ('Thanh',1.0),('Hung',1.0),('Duy',1.0),('Nam',1.0),('Trung',1.0),('Phuc',1.0),('Long',1.0),('Khanh',1.0),
    ('Bao',1.0),('Dat',1.0),('Hai',1.0),('Kien',1.0),('Loc',1.0),('Nghia',1.0),('Phong',1.0),('Quoc',1.0),
    ('Son',1.0),('Thai',1.0),('Truong',1.0),('Vu',1.0),('Xuan',1.0),('Binh',1.0),('Chinh',1.0),('Dinh',1.0),
    ('Giang',1.0),('Hieu',1.0),('Khai',1.0),('Loi',1.0),('Manh',1.0),('Ninh',1.0),('Phat',1.0),('Sang',1.0);

INSERT INTO lastnames_vn (lastname, freq) VALUES
    ('Nguyen',1.0),('Tran',1.0),('Le',1.0),('Pham',1.0),('Hoang',1.0),('Huynh',1.0),('Phan',1.0),
    ('Vu',1.0),('Vo',1.0),('Dang',1.0),('Bui',1.0),('Do',1.0),('Ho',1.0),('Ngo',1.0),
    ('Duong',1.0),('Ly',1.0),('Trinh',1.0),('Dinh',1.0),('Ta',1.0),('Truong',1.0),
    ('Dao',1.0),('Mai',1.0),('Vuong',1.0),('Luu',1.0),('Cao',1.0),('Chu',1.0),('Dam',1.0),
    ('Doan',1.0),('Han',1.0),('Kieu',1.0),('Luong',1.0),('Ma',1.0),('Nong',1.0),('Quach',1.0),
    ('Quan',1.0),('Thai',1.0),('Thao',1.0),('Trang',1.0);

INSERT INTO names_id ("name", freq) VALUES
    ('Egy',1.0),('Witan',1.0),('Asnawi',1.0),('Rachmat',1.0),('Marc',1.0),('Ricky',1.0),('Ezra',1.0),
    ('Rizky',1.0),('Pratama',1.0),('Bagas',1.0),('Ronaldo',1.0),('Muhammad',1.0),('Kevin',1.0),('Diego',1.0),
    ('Beckham',1.0),('Fajar',1.0),('Elkan',1.0),('Yakob',1.0),('Irfan',1.0),('Feby',1.0),
    ('Saddil',1.0),('Osvaldo',1.0),('Andy',1.0),('Iwan',1.0),('Bayu',1.0),('Yudha',1.0),('Dedik',1.0),
    ('Wawan',1.0),('Zulfiandi',1.0),('Sandy',1.0),('Fachruddin',1.0),('Rizki',1.0),
    ('Braif',1.0),('Gian',1.0),('Kadek',1.0),('Rahmat',1.0),('Arya',1.0),('Fikri',1.0);

INSERT INTO lastnames_id (lastname, freq) VALUES
    ('Maulana',1.0),('Wibowo',1.0),('Setiawan',1.0),('Hakim',1.0),('Ramadhan',1.0),('Susanto',1.0),
    ('Utomo',1.0),('Santoso',1.0),('Kurniawan',1.0),('Saputra',1.0),('Nugroho',1.0),('Pratama',1.0),
    ('Widodo',1.0),('Suryanto',1.0),('Gunawan',1.0),('Wijaya',1.0),('Firmansyah',1.0),('Yulianto',1.0),
    ('Iskandar',1.0),('Prasetyo',1.0),('Handoko',1.0),('Purnomo',1.0),('Kusuma',1.0),
    ('Halim',1.0),('Effendi',1.0),('Siregar',1.0),('Sinaga',1.0),('Hutagalung',1.0),('Simanjuntak',1.0),
    ('Panjaitan',1.0),('Nasution',1.0),('Harahap',1.0),('Situmorang',1.0),('Marpaung',1.0),
    ('Sitorus',1.0),('Tampubolon',1.0),('Pardede',1.0),('Silalahi',1.0);

INSERT INTO names_ph ("name", freq) VALUES
    ('Stephan',1.0),('Patrick',1.0),('Angel',1.0),('Iain',1.0),('Amani',1.0),('Justin',1.0),('Manny',1.0),
    ('Simone',1.0),('Kevin',1.0),('Daisuke',1.0),('Mark',1.0),('Jarvey',1.0),('Sandro',1.0),
    ('Miguel',1.0),('Michael',1.0),('Joshua',1.0),('Zico',1.0),('Jefferson',1.0),
    ('Anton',1.0),('Marius',1.0),('Alfred',1.0),('Curt',1.0),('Ray',1.0),('Karl',1.0),('Chad',1.0),
    ('Dylan',1.0),('Julian',1.0),('Nathaniel',1.0),('Christian',1.0),('Gerald',1.0),('Randolf',1.0),
    ('Kaya',1.0),('Robert',1.0),('Ernie',1.0),('Ray',1.0),('Jake',1.0),('Arnold',1.0);

INSERT INTO lastnames_ph (lastname, freq) VALUES
    ('Schrock',1.0),('Reichelt',1.0),('Gonzales',1.0),('Fletcher',1.0),('Guirado',1.0),('Baas',1.0),
    ('Aguinaldo',1.0),('Ott',1.0),('Younghusband',1.0),('Amida',1.0),('Suzuki',1.0),('Hartmann',1.0),
    ('Cabigting',1.0),('Porteria',1.0),('Mora',1.0),('Musico',1.0),('Malicse',1.0),
    ('Etheridge',1.0),('Denmark',1.0),('Yatar',1.0),('Ike',1.0),('Pasilan',1.0),('Manalang',1.0),
    ('Del Rosario',1.0),('Sanvictores',1.0),('Alcantara',1.0),('Miller',1.0),
    ('Villahermosa',1.0),('Dennehy',1.0),('Almira',1.0),('Alegre',1.0),('Marasigan',1.0),
    ('Palla',1.0),('Barbon',1.0),('McDaniel',1.0),('Reyes',1.0);

INSERT INTO names_sa ("name", freq) VALUES
    ('Salem',1.0),('Salman',1.0),('Firas',1.0),('Fahad',1.0),('Abdullah',1.0),('Yasser',1.0),('Nawaf',1.0),
    ('Saud',1.0),('Mohammed',1.0),('Yahya',1.0),('Sultan',1.0),('Nasser',1.0),('Khalid',1.0),
    ('Abdulrahman',1.0),('Turki',1.0),('Faisal',1.0),('Hattan',1.0),('Majed',1.0),('Ziyad',1.0),
    ('Hamzah',1.0),('Sami',1.0),('Waleed',1.0),('Basil',1.0),('Raed',1.0),('Naif',1.0),('Talal',1.0),
    ('Bandar',1.0),('Rakan',1.0),('Musab',1.0),('Ayman',1.0),('Fawaz',1.0),('Hesham',1.0),
    ('Marzouq',1.0),('Osama',1.0),('Qusai',1.0),('Rayan',1.0),('Suhail',1.0),('Thamer',1.0);

INSERT INTO lastnames_sa (lastname, freq) VALUES
    ('Al-Dawsari',1.0),('Al-Faraj',1.0),('Al-Shehri',1.0),('Al-Owais',1.0),('Al-Buraikan',1.0),
    ('Al-Bulaihi',1.0),('Al-Amri',1.0),('Al-Ghannam',1.0),('Al-Malki',1.0),('Al-Hamdan',1.0),
    ('Al-Sahafi',1.0),('Al-Yami',1.0),('Al-Harbi',1.0),('Al-Otaibi',1.0),('Al-Ghamdi',1.0),
    ('Al-Zahrani',1.0),('Al-Qahtani',1.0),('Al-Anazi',1.0),('Al-Enezi',1.0),('Al-Juhani',1.0),
    ('Al-Balawi',1.0),('Al-Mutairi',1.0),('Al-Ahmadi',1.0),('Al-Khaibari',1.0),('Al-Rashidi',1.0),
    ('Al-Subaie',1.0),('Al-Yousef',1.0),('Al-Nasser',1.0),('Al-Fahad',1.0),('Al-Turki',1.0),
    ('Al-Salem',1.0),('Al-Mansour',1.0),('Al-Ammar',1.0),('Al-Jaber',1.0),('Al-Suwailem',1.0),
    ('Al-Hazmi',1.0);

INSERT INTO names_ir ("name", freq) VALUES
    ('Mehdi',1.0),('Sardar',1.0),('Alireza',1.0),('Karim',1.0),('Vahid',1.0),('Saman',1.0),('Ehsan',1.0),
    ('Ashkan',1.0),('Ramin',1.0),('Sepehr',1.0),('Milad',1.0),('Omid',1.0),('Reza',1.0),('Ali',1.0),
    ('Hossein',1.0),('Mohammad',1.0),('Amir',1.0),('Payam',1.0),('Morteza',1.0),('Shoja',1.0),
    ('Farshad',1.0),('Danial',1.0),('Roozbeh',1.0),('Saeid',1.0),('Hamid',1.0),('Kaveh',1.0),
    ('Bahram',1.0),('Arash',1.0),('Babak',1.0),('Cyrus',1.0),('Dariush',1.0),('Farhad',1.0),
    ('Hooman',1.0),('Iman',1.0),('Javad',1.0),('Kian',1.0),('Nima',1.0),('Peyman',1.0),('Rostam',1.0);

INSERT INTO lastnames_ir (lastname, freq) VALUES
    ('Taremi',1.0),('Azmoun',1.0),('Jahanbakhsh',1.0),('Ansarifard',1.0),('Hosseini',1.0),
    ('Pouraliganji',1.0),('Moharrami',1.0),('Shojaei',1.0),('Ezatolahi',1.0),('Karimi',1.0),
    ('Rezaeian',1.0),('Ghoddos',1.0),('Amiri',1.0),('Cheshmi',1.0),('Hajsafi',1.0),
    ('Nourollahi',1.0),('Rafiei',1.0),('Sadeghi',1.0),('Hashemian',1.0),
    ('Daei',1.0),('Bagheri',1.0),('Khalatbari',1.0),('Mansourian',1.0),('Nekounam',1.0),
    ('Teymourian',1.0),('Kalantari',1.0),('Madanchi',1.0),('Motahari',1.0),
    ('Naderi',1.0),('Piroozi',1.0),('Rahnama',1.0),('Sabet',1.0),('Taghavi',1.0),('Vali',1.0),
    ('Zamani',1.0),('Askari',1.0),('Bayat',1.0),('Farahani',1.0);

-- Africa: Egypt, Ghana, Ivory Coast, Cameroon, Senegal, South Africa, Morocco, Algeria, Kenya

INSERT INTO names_eg ("name", freq) VALUES
    ('Mohamed',1.0),('Mahmoud',1.0),('Ahmed',1.0),('Omar',1.0),('Amr',1.0),('Hossam',1.0),('Karim',1.0),
    ('Sherif',1.0),('Ramadan',1.0),('Marwan',1.0),('Emam',1.0),('Ayman',1.0),('Tarek',1.0),
    ('Hassan',1.0),('Hany',1.0),('Nabil',1.0),('Sameh',1.0),('Osama',1.0),('Wael',1.0),('Abdallah',1.0),
    ('Yasser',1.0),('Ibrahim',1.0),('Khaled',1.0),('Islam',1.0),('Mostafa',1.0),('Akram',1.0),
    ('Bassem',1.0),('Fathy',1.0),('Gamal',1.0),('Hazem',1.0),('Ihab',1.0),('Jamal',1.0),
    ('Kareem',1.0),('Loay',1.0),('Magdy',1.0),('Nasser',1.0),('Rami',1.0),('Salah',1.0);

INSERT INTO lastnames_eg (lastname, freq) VALUES
    ('El-Nenny',1.0),('Elneny',1.0),('Elmohamady',1.0),('Hegazi',1.0),('Fathy',1.0),
    ('El-Shenawy',1.0),('Marmoush',1.0),('Ashour',1.0),('Kahraba',1.0),('Ramadan',1.0),
    ('Soliman',1.0),('Salah',1.0),('Kamal',1.0),('Abdelmonem',1.0),('Hamdy',1.0),('Sabry',1.0),
    ('Farag',1.0),('Mahmoud',1.0),('Ibrahim',1.0),('Kamel',1.0),('Fouad',1.0),
    ('Rashad',1.0),('Shawky',1.0),('Wahba',1.0),('Yassin',1.0),('Amer',1.0),('Badr',1.0),
    ('Dessouki',1.0),('Emara',1.0),('Farid',1.0),('Ghali',1.0),('Hosny',1.0),('Kassem',1.0),
    ('Mansour',1.0),('Naguib',1.0),('Rifai',1.0),('Sultan',1.0);

INSERT INTO names_gh ("name", freq) VALUES
    ('Thomas',1.0),('Andre',1.0),('Jordan',1.0),('Mohammed',1.0),('Iddrisu',1.0),('Baba',1.0),
    ('Osman',1.0),('Alexander',1.0),('Daniel',1.0),('Joseph',1.0),('Jonathan',1.0),('Richmond',1.0),
    ('Christopher',1.0),('Emmanuel',1.0),('Samuel',1.0),('Benjamin',1.0),('Isaac',1.0),('Prince',1.0),
    ('Kwame',1.0),('Kofi',1.0),('Kwabena',1.0),('Yaw',1.0),('Kwaku',1.0),('Nana',1.0),
    ('Ebenezer',1.0),('Fiifi',1.0),('Gideon',1.0),('Ishmael',1.0),('Justice',1.0),('Lawrence',1.0),
    ('Michael',1.0),('Nathaniel',1.0),('Obed',1.0),('Patrick',1.0),('Quincy',1.0),('Solomon',1.0),
    ('Vincent',1.0),('Wisdom',1.0);

INSERT INTO lastnames_gh (lastname, freq) VALUES
    ('Partey',1.0),('Ayew',1.0),('Amartey',1.0),('Salisu',1.0),('Kudus',1.0),('Baba',1.0),
    ('Semenyo',1.0),('Sarkodie',1.0),('Djiku',1.0),('Mensah',1.0),('Owusu',1.0),('Boateng',1.0),
    ('Asante',1.0),('Appiah',1.0),('Adjei',1.0),('Amoah',1.0),('Osei',1.0),('Yeboah',1.0),
    ('Frimpong',1.0),('Acheampong',1.0),('Antwi',1.0),('Bediako',1.0),('Danso',1.0),('Essien',1.0),
    ('Gyan',1.0),('Gyasi',1.0),('Kwarteng',1.0),('Larbi',1.0),('Ntiamoah',1.0),('Opoku',1.0),
    ('Quansah',1.0),('Sarpong',1.0),('Tetteh',1.0),('Twumasi',1.0),('Wiredu',1.0),('Yankey',1.0),
    ('Addo',1.0),('Baah',1.0),('Dapaah',1.0),('Frempong',1.0);

INSERT INTO names_ci ("name", freq) VALUES
    ('Didier',1.0),('Yaya',1.0),('Kolo',1.0),('Wilfried',1.0),('Serge',1.0),('Franck',1.0),('Nicolas',1.0),
    ('Ibrahim',1.0),('Eric',1.0),('Odilon',1.0),('Simon',1.0),('Cheick',1.0),('Serey',1.0),
    ('Ghislain',1.0),('Amad',1.0),('Sebastien',1.0),('Willy',1.0),('Christian',1.0),
    ('Boubacar',1.0),('Ismael',1.0),('Souleymane',1.0),('Karim',1.0),('Mory',1.0),('Adama',1.0),
    ('Bafetimbi',1.0),('Emmanuel',1.0),('Gervinho',1.0),('Guy',1.0),('Hamed',1.0),('Jonathan',1.0),
    ('Lacina',1.0),('Mamadou',1.0),('Ousmane',1.0),('Salomon',1.0),('Siaka',1.0),('Tiemoue',1.0),('Zie',1.0);

INSERT INTO lastnames_ci (lastname, freq) VALUES
    ('Drogba',1.0),('Toure',1.0),('Zaha',1.0),('Gradel',1.0),('Aurier',1.0),('Kessie',1.0),
    ('Fofana',1.0),('Aholou',1.0),('Sangare',1.0),('Konan',1.0),('Kone',1.0),('Bailly',1.0),
    ('Traore',1.0),('Coulibaly',1.0),('Diomande',1.0),('Doumbia',1.0),('Sanogo',1.0),('Diakite',1.0),
    ('Kouassi',1.0),('Yao',1.0),('Kouame',1.0),('Ouattara',1.0),('Bamba',1.0),('Diarra',1.0),
    ('Kobenan',1.0),('Meite',1.0),('Pokou',1.0),('Silue',1.0),('Tanoh',1.0),
    ('Yapi',1.0),('Zadi',1.0),('Angoua',1.0),('Bakayoko',1.0),('Diabate',1.0),('Kanga',1.0),('Yeo',1.0);

INSERT INTO names_cm ("name", freq) VALUES
    ('Samuel',1.0),('Andre',1.0),('Vincent',1.0),('Eric',1.0),('Joel',1.0),('Nicolas',1.0),('Karl',1.0),
    ('Christian',1.0),('Jean',1.0),('Bryan',1.0),('Martin',1.0),('Ambroise',1.0),('Yves',1.0),
    ('Michael',1.0),('Enzo',1.0),('Nouhou',1.0),('Olivier',1.0),('Pierre',1.0),('Frank',1.0),
    ('Georges',1.0),('Herve',1.0),('Jerome',1.0),('Landry',1.0),('Marius',1.0),('Norbert',1.0),
    ('Patrice',1.0),('Rene',1.0),('Serge',1.0),('Thierry',1.0),('Achille',1.0),('Bertrand',1.0),
    ('Cedric',1.0),('Didier',1.0),('Emile',1.0),('Florent',1.0),('Guy',1.0),('Hilaire',1.0),
    ('Innocent',1.0),('Josue',1.0);

INSERT INTO lastnames_cm (lastname, freq) VALUES
    ('Etoo',1.0),('Song',1.0),('Aboubakar',1.0),('Onana',1.0),('Fai',1.0),('Zambo',1.0),
    ('Ngamaleu',1.0),('Anguissa',1.0),('Choupo-Moting',1.0),('Bassogog',1.0),('Mbeumo',1.0),('Ekambi',1.0),
    ('Kunde',1.0),('Mbekeli',1.0),('Ondoua',1.0),('Wooh',1.0),('Castelletto',1.0),
    ('Ngoumou',1.0),('Nkoulou',1.0),('Bahoken',1.0),('Banana',1.0),('Bikoro',1.0),('Ekanga',1.0),
    ('Ekwalla',1.0),('Foe',1.0),('Kome',1.0),('Manga',1.0),('Meyong',1.0),('Ndip',1.0),
    ('Nkoudou',1.0),('Nnomo',1.0),('Nsame',1.0),('Ottou',1.0),('Owona',1.0),('Poka',1.0),
    ('Talom',1.0),('Yombi',1.0);

INSERT INTO names_sn ("name", freq) VALUES
    ('Sadio',1.0),('Kalidou',1.0),('Idrissa',1.0),('Ismaila',1.0),('Cheikhou',1.0),('Edouard',1.0),
    ('Bouna',1.0),('Salif',1.0),('Papa',1.0),('Krepin',1.0),('Nampalys',1.0),('Boulaye',1.0),
    ('Habib',1.0),('Youssouf',1.0),('Moussa',1.0),('Abdou',1.0),('Alfred',1.0),('Amath',1.0),
    ('Badou',1.0),('Cheikh',1.0),('Demba',1.0),('Famara',1.0),('Gana',1.0),('Habibou',1.0),
    ('Ibou',1.0),('Khadim',1.0),('Lamine',1.0),('Mame',1.0),('Ndiaga',1.0),('Oumar',1.0),
    ('Pathe',1.0),('Racine',1.0),('Saliou',1.0),('Tapha',1.0),('Youssoupha',1.0),('Aliou',1.0),('Bacary',1.0);

INSERT INTO lastnames_sn (lastname, freq) VALUES
    ('Mane',1.0),('Koulibaly',1.0),('Gueye',1.0),('Sarr',1.0),('Kouyate',1.0),('Mendy',1.0),
    ('Sabaly',1.0),('Diallo',1.0),('Sy',1.0),('Cisse',1.0),('Ciss',1.0),('Faye',1.0),
    ('Diagne',1.0),('Dieng',1.0),('Fall',1.0),('Ndiaye',1.0),('Thiam',1.0),('Ba',1.0),
    ('Diatta',1.0),('Diedhiou',1.0),('Diop',1.0),('Fofana',1.0),('Gomis',1.0),('Jakobs',1.0),
    ('Kane',1.0),('Konate',1.0),('Ndao',1.0),('Niang',1.0),('Sane',1.0),('Seck',1.0),
    ('Sonko',1.0),('Sow',1.0),('Thioune',1.0),('Wague',1.0),('Balde',1.0),('Toure',1.0);

INSERT INTO names_za ("name", freq) VALUES
    ('Percy',1.0),('Themba',1.0),('Bongani',1.0),('Thulani',1.0),('Siphiwe',1.0),('Itumeleng',1.0),
    ('Andile',1.0),('Siyabonga',1.0),('Lebogang',1.0),('Thembinkosi',1.0),('Ronwen',1.0),('Keagan',1.0),
    ('Sipho',1.0),('Vincent',1.0),('Innocent',1.0),('Hlompho',1.0),('Rulani',1.0),('Jabu',1.0),
    ('Katlego',1.0),('Lyle',1.0),('Mothobi',1.0),('Njabulo',1.0),('Oswin',1.0),('Peter',1.0),
    ('Quinton',1.0),('Rowen',1.0),('Steven',1.0),('Teboho',1.0),('Wandile',1.0),('Aubrey',1.0),
    ('Bafana',1.0),('Charlton',1.0),('Dumisani',1.0),('Elias',1.0),('Fortune',1.0),('Given',1.0),
    ('Hendrick',1.0),('Isaac',1.0),('Jomo',1.0);

INSERT INTO lastnames_za (lastname, freq) VALUES
    ('Tau',1.0),('Zwane',1.0),('Zungu',1.0),('Dolly',1.0),('Foster',1.0),('Mokoena',1.0),
    ('Sirino',1.0),('Khune',1.0),('Mokwena',1.0),('Ndlovu',1.0),('Modise',1.0),('Ngcongca',1.0),
    ('Kekana',1.0),('Mahlangu',1.0),('Masilela',1.0),('Ngobese',1.0),('Rantie',1.0),('Serero',1.0),
    ('Shalulile',1.0),('Sithole',1.0),('Xulu',1.0),('Zulu',1.0),('Baloyi',1.0),('Chabalala',1.0),
    ('Dladla',1.0),('Gumede',1.0),('Hlatshwayo',1.0),('Jali',1.0),('Khumalo',1.0),
    ('Maluleke',1.0),('Nhlapo',1.0),('Nyathi',1.0),('Radebe',1.0),('Sikhosana',1.0),('Sithebe',1.0),
    ('Tshabalala',1.0),('Vilakazi',1.0),('Xoki',1.0);

INSERT INTO names_ma ("name", freq) VALUES
    ('Achraf',1.0),('Sofyan',1.0),('Hakim',1.0),('Youssef',1.0),('Noussair',1.0),('Yassine',1.0),
    ('Ayoub',1.0),('Romain',1.0),('Nayef',1.0),('Zakaria',1.0),('Selim',1.0),('Bilal',1.0),
    ('Ilias',1.0),('Munir',1.0),('Adam',1.0),('Amine',1.0),('Badr',1.0),('Chamseddine',1.0),
    ('Driss',1.0),('Ezzine',1.0),('Farid',1.0),('Ghali',1.0),('Hamza',1.0),('Ismail',1.0),
    ('Jamal',1.0),('Karim',1.0),('Lahcen',1.0),('Mehdi',1.0),('Nabil',1.0),('Omar',1.0),
    ('Qassim',1.0),('Rachid',1.0),('Said',1.0),('Tarik',1.0),('Walid',1.0),('Yassir',1.0),
    ('Zaid',1.0),('Aymane',1.0),('Brahim',1.0),('Chakib',1.0);

INSERT INTO lastnames_ma (lastname, freq) VALUES
    ('Hakimi',1.0),('Ziyech',1.0),('En-Nesyri',1.0),('Boufal',1.0),('Mazraoui',1.0),('Saiss',1.0),
    ('Aguerd',1.0),('Ounahi',1.0),('Amrabat',1.0),('Bounou',1.0),('Dari',1.0),('Cheddira',1.0),
    ('Zabiri',1.0),('El Kaabi',1.0),('Benoun',1.0),('Fajr',1.0),('Hamdallah',1.0),('Jabrane',1.0),
    ('Loutfi',1.0),('Masrour',1.0),('Nassiri',1.0),('Ouahbi',1.0),('Regragui',1.0),('Sabiri',1.0),
    ('Tagnaouti',1.0),('Wahbi',1.0),('Aboukhlal',1.0),('Belhanda',1.0),('Chamakh',1.0),('Dirar',1.0),
    ('El Ahmadi',1.0),('Fellaini',1.0),('Ghanam',1.0),('Harit',1.0),('Idrissi',1.0),('Jbari',1.0),
    ('Karam',1.0),('Lamrissi',1.0),('Meski',1.0);

INSERT INTO names_dz ("name", freq) VALUES
    ('Riyad',1.0),('Ismael',1.0),('Islam',1.0),('Baghdad',1.0),('Yacine',1.0),('Sofiane',1.0),
    ('Youcef',1.0),('Ramy',1.0),('Rachid',1.0),('Aissa',1.0),('Djamel',1.0),('Ahmed',1.0),
    ('Nabil',1.0),('Sid',1.0),('Adem',1.0),('Bilal',1.0),('Chems',1.0),('Farid',1.0),
    ('Hocine',1.0),('Ilyes',1.0),('Karim',1.0),('Larbi',1.0),('Mehdi',1.0),('Nassim',1.0),
    ('Omar',1.0),('Rabah',1.0),('Salim',1.0),('Tayeb',1.0),('Walid',1.0),('Yassine',1.0),
    ('Zakaria',1.0),('Abdelkader',1.0),('Brahim',1.0),('Cherif',1.0),('Djaffar',1.0),('Fouad',1.0),
    ('Hakim',1.0),('Idir',1.0),('Jugurtha',1.0);

INSERT INTO lastnames_dz (lastname, freq) VALUES
    ('Mahrez',1.0),('Slimani',1.0),('Bennacer',1.0),('Belaili',1.0),('Delandri',1.0),('Feghouli',1.0),
    ('Bounedjah',1.0),('Boudaoui',1.0),('Benlamri',1.0),('Atal',1.0),('Mandi',1.0),('Brahimi',1.0),
    ('Belkebla',1.0),('Guedioura',1.0),('Bentaleb',1.0),('Zerrouki',1.0),('Bensebaini',1.0),('Kacher',1.0),
    ('Aouar',1.0),('Boudebouz',1.0),('Zorgane',1.0),('Yahia',1.0),('Bezzaz',1.0),('Chetti',1.0),
    ('Djabou',1.0),('Ferhat',1.0),('Ghezzal',1.0),('Hanni',1.0),('Ighil',1.0),('Kadir',1.0),
    ('Meziane',1.0),('Naidji',1.0),('Ounas',1.0),('Rekik',1.0),('Saidi',1.0),('Taider',1.0),
    ('Younes',1.0),('Zaidi',1.0);

INSERT INTO names_ke ("name", freq) VALUES
    ('Michael',1.0),('Victor',1.0),('Johanna',1.0),('Musa',1.0),('Erick',1.0),('Francis',1.0),('Duke',1.0),
    ('Timothy',1.0),('Boniface',1.0),('Eric',1.0),('Joseph',1.0),('Lawrence',1.0),('Ayub',1.0),
    ('Clifton',1.0),('Enock',1.0),('Kepha',1.0),('Masoud',1.0),('Vincent',1.0),
    ('Abud',1.0),('Bernard',1.0),('Collins',1.0),('David',1.0),('Elijah',1.0),('Felix',1.0),
    ('Gilbert',1.0),('Harun',1.0),('Isaiah',1.0),('Jackson',1.0),('Kelvin',1.0),('Levis',1.0),
    ('Moses',1.0),('Nicholas',1.0),('Oscar',1.0),('Patrick',1.0),('Robert',1.0),('Samuel',1.0),
    ('Titus',1.0),('Wesley',1.0);

INSERT INTO lastnames_ke (lastname, freq) VALUES
    ('Wanyama',1.0),('Olunga',1.0),('Ayimba',1.0),('Origi',1.0),('Mariga',1.0),('Otieno',1.0),
    ('Odhiambo',1.0),('Onyango',1.0),('Wafula',1.0),('Kioko',1.0),('Mutiso',1.0),('Wekesa',1.0),
    ('Simiyu',1.0),('Odongo',1.0),('Ochieng',1.0),('Njoroge',1.0),('Kamau',1.0),('Mwangi',1.0),
    ('Kariuki',1.0),('Njenga',1.0),('Kimani',1.0),('Karanja',1.0),('Waweru',1.0),('Gitau',1.0),
    ('Macharia',1.0),('Wachira',1.0),('Njuguna',1.0),('Ndungu',1.0),('Kigen',1.0),('Rotich',1.0),
    ('Kiprop',1.0),('Cheruiyot',1.0),('Korir',1.0),('Ruto',1.0),('Sang',1.0),('Barasa',1.0),
    ('Wamalwa',1.0),('Nyongesa',1.0);

-- World structure config: the tier/group/teams-per-league shape used the next time leagues are
-- (re)created (see common.GetWorldConfig/SetWorldConfig, handlers.createCountryLeagues).
-- Singleton row - id is always 1.
CREATE TABLE world_config (
    id    smallint PRIMARY KEY DEFAULT 1,
    tiers jsonb NOT NULL,
    CONSTRAINT world_config_singleton CHECK (id = 1)
);

-- Per-league match-day/time override. A league with no row here uses the app-wide default (the
-- season table's match_schedule_* columns, see common.GetMatchSchedule) - see
-- common.GetLeagueScheduleOverrides and engine.GenerateMatches.
CREATE TABLE league_schedule (
    league_id     bigint PRIMARY KEY REFERENCES leagues(league_id) ON DELETE CASCADE,
    weekdays_mask int NOT NULL,
    hour          int NOT NULL,
    minute        int NOT NULL
);

-- A team's head coach - hired from an open market of generated candidates (see
-- common.GenerateTrainerCandidates), at most one per team. A hired trainer's skill accelerates
-- player attribute growth and raises the youth academy's prospect ceiling (see
-- engine.ProgressPlayers) and costs a wage deducted alongside player wages after each match (see
-- engine.deductTrainerWage). team_id NULL means the trainer is on the open market, unemployed -
-- the same convention players.team_id uses for free agents.
CREATE TABLE trainers (
    trainer_id bigserial PRIMARY KEY,
    first_name varchar(100) NOT NULL,
    last_name varchar(100) NOT NULL,
    team_id bigint REFERENCES teams(team_id) ON DELETE SET NULL,
    skill int NOT NULL,
    wage int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Friendly matches: a team can challenge any other team to a non-competitive match that the
-- other side must accept before it's scheduled - no effect on league classification or cup
-- progression, no ticket revenue/sponsorship, no AI-team upkeep hook (see engine/friendlies.go).
-- Deliberately NOT built on the matches/match_events tables, same reasoning as cup_ties: those
-- are deeply wired to league classification, so a different kind of match gets its own tables.
CREATE TABLE friendly_challenges (
    challenge_id        bigserial PRIMARY KEY,
    challenger_team_id  bigint NOT NULL REFERENCES teams(team_id),
    opponent_team_id    bigint NOT NULL REFERENCES teams(team_id),
    created_at          timestamptz NOT NULL DEFAULT now(),
    -- The kickoff time the challenger optionally proposed when sending the challenge (see
    -- common.ChallengeToFriendly) - null means they left it up to the opponent. The opponent
    -- still has final say when accepting; this is only ever used as a suggested default, never
    -- enforced.
    proposed_kickoff    timestamptz,
    -- Which side of the fixture the challenger takes once accepted (see
    -- common.acceptFriendlyChallenge) - true puts the challenger at home, false makes the
    -- opponent the home team instead. Added by init62_friendly_challenge_home_side.sql.
    challenger_is_home  boolean NOT NULL DEFAULT true,
    UNIQUE (challenger_team_id, opponent_team_id)
);

CREATE TABLE friendly_matches (
    match_id       bigserial PRIMARY KEY,
    home_team_id   bigint NOT NULL REFERENCES teams(team_id),
    away_team_id   bigint NOT NULL REFERENCES teams(team_id),
    match_date     timestamptz NOT NULL,
    played         boolean NOT NULL DEFAULT false,
    in_simulation  boolean NOT NULL DEFAULT false,
    home_goals     int NOT NULL DEFAULT 0,
    away_goals     int NOT NULL DEFAULT 0
);

CREATE TABLE friendly_events (
    event_id            bigserial PRIMARY KEY,
    match_id            bigint NOT NULL REFERENCES friendly_matches(match_id),
    event_type          int,
    team_owner          bigint REFERENCES teams(team_id),
    other_team          bigint REFERENCES teams(team_id),
    minute              int,
    team_players        varchar(200),
    team_player_names   varchar(1000),
    home_goals          int DEFAULT 0,
    away_goals          int DEFAULT 0
);

-- Per-minute zone-by-zone team-strength snapshots for a played friendly (see common.MatchInstant
-- / engine.runMatchSimulation) - the friendly equivalent of match_instants, kept in its own table
-- for the same reason friendly_lineup is separate from lineup: friendly_matches has its own id
-- sequence, so writing here to match_instants would either violate its foreign key to
-- matches(match_id) or silently collide with an unrelated league match sharing the same id.
CREATE TABLE friendly_instants (
    match_id bigint not null references friendly_matches(match_id),
    minute int not null,
    home_goals int default 0,
    away_goals int default 0,
    home_qua varchar(500),
    away_qua varchar(500),
    midfield_diff double precision,
    homeattack_right_diff double precision,
    homeattack_left_diff double precision,
    homeattack_center_diff double precision,
    awayattack_right_diff double precision,
    awayattack_left_diff double precision,
    awayattack_center_diff double precision,
    primary key (match_id, minute)
);

-- Per-match lineup override for a friendly, so a manager can set a different XI for each
-- upcoming friendly instead of always playing their team's standing default lineup - see
-- common.GetLineupsForFriendly, common.SetFriendlyLineup, handlers.SetFriendlyLineupHandler.
--
-- Deliberately NOT built on `lineup`/`lineup_positions`: those are keyed by `match_id` with a
-- foreign key to `matches(match_id)` (competitive league matches only) - friendly_matches has its
-- own id sequence, so writing a friendly's override there would either violate that foreign key
-- or, worse, silently collide with an unrelated competitive match that happens to share the same
-- id. A friendly gets its own, identically-shaped pair of tables instead, so this can never
-- happen. formation is stored per friendly (rather than always reusing the team's standing
-- formation - see teams.formation) so a manager can pick a different shape for one friendly
-- without changing anything about their competitive lineup.
CREATE TABLE friendly_lineup (
    lineup_id bigserial PRIMARY KEY,
    match_id  bigint NOT NULL REFERENCES friendly_matches(match_id),
    team_id   bigint NOT NULL REFERENCES teams(team_id),
    formation smallint NOT NULL DEFAULT 0,
    UNIQUE (match_id, team_id)
);

CREATE TABLE friendly_lineup_positions (
    lineup_id bigint NOT NULL REFERENCES friendly_lineup(lineup_id),
    player_id bigint REFERENCES players(player_id) ON DELETE SET NULL,
    position  smallint NOT NULL
);

-- A manager's planned in-match substitutions for their default lineup - up to 3, each swapping
-- out a current starter for a bench player at a chosen minute (see
-- common.SetDefaultLineupSubstitutions, engine.SubstitutionSimulation). slot is 1-3, assigned by
-- save order (not user-chosen) - just enough to give each of a team's up-to-3 planned
-- substitutions its own row under the (team_id, slot) primary key.
CREATE TABLE default_lineup_substitutions (
    team_id       bigint NOT NULL REFERENCES teams(team_id),
    slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 3),
    player_out_id bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    player_in_id  bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    minute        smallint NOT NULL CHECK (minute BETWEEN 1 AND 90),
    PRIMARY KEY (team_id, slot)
);

-- One row per player per season progression run (see engine.ProgressPlayers) - captures the
-- before/after age and seven core skills at the exact moment they change, since the drift itself
-- is randomized (see engine.driftSkill) and can't be recomputed after the fact. Backs the player
-- details page's "latest evolution" section and the squad-wide "latest evolutions" view linked
-- from its notification. ON DELETE CASCADE because RetirePlayer hard-deletes the player row (see
-- common.RetirePlayer) - once a player is gone there's no profile page left to show this on.
CREATE TABLE player_progression_log (
    id bigserial PRIMARY KEY,
    player_id bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    season integer NOT NULL,
    age_before integer NOT NULL,
    age_after integer NOT NULL,
    goal_keeper_before integer NOT NULL,
    goal_keeper_after integer NOT NULL,
    defense_before integer NOT NULL,
    defense_after integer NOT NULL,
    lateral_before integer NOT NULL,
    lateral_after integer NOT NULL,
    pass_before integer NOT NULL,
    pass_after integer NOT NULL,
    midfield_before integer NOT NULL,
    midfield_after integer NOT NULL,
    dribbling_before integer NOT NULL,
    dribbling_after integer NOT NULL,
    shot_before integer NOT NULL,
    shot_after integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Indexes carried over unchanged from the original schema.
create index classification_idx1 on classification (league_id);
create index classification_idx2 on classification(team_id);

create index matches_idx1 on matches(match_date desc);
create index matches_idx2 on matches(league_id, week_number);

create index match_instants_idx1 on match_instants(match_id);

create index teams_idx1 on teams(country);
create index teams_idx2 on teams(owner_id);

CREATE INDEX team_transactions_idx1 ON team_transactions(team_id, created_at DESC);
-- Also serves common.GetPlayerTransactions (WHERE player_id = $1 ORDER BY created_at DESC),
-- mirroring the (team_id, created_at DESC) index above.
CREATE INDEX team_transactions_player_idx ON team_transactions(player_id, created_at DESC) WHERE player_id IS NOT NULL;

CREATE INDEX transfer_listings_idx1 ON transfer_listings(selling_team_id);

CREATE INDEX cup_ties_idx1 ON cup_ties(cup_id, round);
CREATE INDEX cup_tie_events_idx1 ON cup_tie_events(tie_id);

CREATE INDEX notifications_idx1 ON notifications(user_id, created_at DESC);

CREATE INDEX loan_listings_idx1 ON loan_listings(lending_team_id);
CREATE INDEX player_loans_idx1 ON player_loans(return_season);

CREATE INDEX transfer_offers_idx1 ON transfer_offers(to_team_id);
CREATE INDEX transfer_offers_idx2 ON transfer_offers(from_team_id);

CREATE INDEX season_history_idx1 ON season_history(team_id, season DESC);

CREATE INDEX group_members_idx1 ON group_members(user_id);

CREATE UNIQUE INDEX trainers_team_unique_idx ON trainers(team_id) WHERE team_id IS NOT NULL;
CREATE INDEX trainers_available_idx ON trainers(skill) WHERE team_id IS NULL;

CREATE INDEX friendly_challenges_opponent_idx ON friendly_challenges(opponent_team_id);
CREATE INDEX friendly_matches_home_idx ON friendly_matches(home_team_id, match_date);
CREATE INDEX friendly_matches_away_idx ON friendly_matches(away_team_id, match_date);
CREATE INDEX friendly_events_idx ON friendly_events(match_id);
CREATE INDEX friendly_lineup_positions_idx1 ON friendly_lineup_positions(lineup_id);

-- New indexes added by this consolidation, identified by cross-referencing every *StmtSql query
-- in backend/common and backend/handlers against the indexes above - each of these backs a real,
-- currently-unindexed WHERE/JOIN clause that runs on a normal request path, not just at
-- migration/admin time:
--
-- players.team_id: the squad list (GetPlayersFromTeamStmtSql), wage bill, player count and
-- contract-renewal ownership checks all filter/join on it, and it had no index at all.
CREATE INDEX players_team_idx ON players(team_id);
-- match_events.match_id: every match view re-fetches its full event timeline by match_id
-- (GetMatchEventsStmtSql) with no supporting index.
CREATE INDEX match_events_match_idx ON match_events(match_id);
-- match_events(event_type, team_players): backs countPlayerEventStmtSql, run for every player
-- details page to count a specific event type credited to that player.
CREATE INDEX match_events_scorer_idx ON match_events(event_type, team_players);
-- lineup_positions had no primary key or index of any kind; GetLineupsForMatchStmtSql joins it
-- to `lineup` on lineup_id for every match simulated.
CREATE INDEX lineup_positions_idx1 ON lineup_positions(lineup_id);
-- matches.home_team_id / away_team_id: common.GetHeadToHead filters on both columns in either
-- direction with no supporting index, forcing a full scan of the matches table.
CREATE INDEX matches_home_team_idx ON matches(home_team_id);
CREATE INDEX matches_away_team_idx ON matches(away_team_id);

-- Indexes for the tables/columns folded in from init38_match_lineup_overrides.sql through
-- init50_transfer_market_stats.sql.
CREATE INDEX cup_tie_lineup_positions_idx1 ON cup_tie_lineup_positions(lineup_id);
-- Guards against two players ever ending up with the same shirt number on the same team - belt
-- and braces alongside the application-level lowest-free-number assignment, which isn't itself
-- immune to a race between two concurrent squad changes for the same team.
CREATE UNIQUE INDEX players_team_shirt_number_idx ON players (team_id, shirt_number)
    WHERE team_id IS NOT NULL AND shirt_number IS NOT NULL;
CREATE INDEX idx_player_progression_log_player_id ON player_progression_log(player_id, id DESC);
CREATE INDEX idx_push_subscriptions_user_id ON push_subscriptions(user_id);
CREATE INDEX idx_player_club_history_player_id ON player_club_history(player_id);
CREATE INDEX friendly_instants_idx ON friendly_instants(match_id);
