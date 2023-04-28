CREATE TABLE public.lastnames_es (
     lastname varchar(100) NULL,
     freq float8 NULL
);

CREATE TABLE public.names_es (
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
   PRIMARY KEY (user_id)
);


CREATE TABLE teams (
   team_id 	bigserial NOT NULL,
   owner_id 	bigint ,
   country		varchar(4) references countries(name),
   league_id	bigint default -1,
   name varchar(200) NULL,
   PRIMARY KEY (team_id)
);

CREATE TABLE season (
    current_season int
);

CREATE TABLE leagues (
     league_id 	bigserial NOT null,
     country	varchar(4) not null references countries(name),
     group_number	int not null,
     name 		varchar(10) not null,
     PRIMARY KEY (league_id)
);


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
     PRIMARY KEY (player_id)
);

CREATE TABLE matches (
     match_id bigserial NOT null PRIMARY KEY,
     league_id int8 NOT NULL REFERENCES leagues(league_id),
     season int not null,
     played bool default false,
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


create index classification_idx1 on classification (league_id);
create index classification_idx2 on classification(team_id);

create index matches_idx1 on matches(match_date desc);
create index matches_idx2 on matches(league_id, week_number);

create index teams_idx1 on teams(country);
create index teams_idx2 on teams(owner_id);


