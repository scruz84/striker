-- Full baseline schema for a fresh database. This file used to be just the original tables,
-- with every later change shipped as its own resources/sql/initN_*.sql migration; those have
-- since been folded in here (most recently init30_player_birth_country.sql through
-- init36_transfer_offer_terms.sql) so a new database only needs to run this one file (plus the
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
   stadium_capacity integer NOT NULL DEFAULT 2000,
   ticket_price integer NOT NULL DEFAULT 10,
   formation smallint NOT NULL DEFAULT 0,
   name_changed boolean NOT NULL DEFAULT false,
   ai_level smallint NOT NULL DEFAULT 0,
   -- Enforces the one-promotion-per-real-calendar-week cap on academy promotions (see
   -- common.CanPromoteFromAcademy) - a single timestamp is enough since at most one promotion can
   -- ever be "the most recent" for a team.
   last_academy_promotion_at timestamptz,
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
    free_agent_injections_per_week smallint NOT NULL DEFAULT 5
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
     away_goals  int default 0
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
CREATE TABLE cups (
    cup_id           bigserial PRIMARY KEY,
    country          varchar(4) NOT NULL REFERENCES countries(name),
    season           int NOT NULL,
    name             varchar(200) NOT NULL,
    completed        boolean NOT NULL DEFAULT false,
    champion_team_id bigint REFERENCES teams(team_id)
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

-- Per-user notification log (match results, promotion/relegation, transfers, cup results). See
-- common/notifications.go for the generation points and engine wiring.
CREATE TABLE notifications (
    notification_id   bigserial PRIMARY KEY,
    user_id            bigint NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    notification_type  smallint NOT NULL,
    message            varchar(1000) NOT NULL,
    read               boolean NOT NULL DEFAULT false,
    created_at         timestamptz NOT NULL DEFAULT now()
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

-- Name pools for more seeded countries (en/fr/de/it/pt), mirroring the names_es/lastnames_es
-- schema. Curated common first/last names (~60 each) rather than a dump the size of
-- names_es/lastnames_es (~29k rows) - a deliberately smaller pool, disclosed as a scope cut: more
-- repetition across a large squad of teams than the Spanish pool has, but real, common names.
-- it/pt are NOT inserted into `countries`/seededCountries here - they stay inactive until an
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

-- Accumulates each player's minutes played at each tactical position across the current
-- season's competitive matches (league + cup - deliberately NOT friendlies). Consumed and
-- cleared once per season by engine.ProgressPlayers, which gates skill growth on having actually
-- played, restricted to the skills relevant to whichever position a player spent the most
-- minutes at, scaled by how many minutes they played in total.
CREATE TABLE player_position_minutes (
    player_id bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    position  smallint NOT NULL,
    minutes   int NOT NULL DEFAULT 0,
    PRIMARY KEY (player_id, position)
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
