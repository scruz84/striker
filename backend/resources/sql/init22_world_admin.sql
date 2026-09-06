-- Admin "world management" tooling: a configurable league structure, per-league match schedule
-- overrides, and two more supported-but-inactive countries an admin can activate on demand.

-- World structure config: the tier/group/teams-per-league shape used the next time leagues are
-- (re)created (see common.GetWorldConfig/SetWorldConfig, handlers.createCountryLeagues).
-- Singleton row - id is always 1.
CREATE TABLE IF NOT EXISTS world_config (
    id    smallint PRIMARY KEY DEFAULT 1,
    tiers jsonb NOT NULL,
    CONSTRAINT world_config_singleton CHECK (id = 1)
);

-- Per-league match-day/time override. A league with no row here uses the app-wide default (the
-- season table's match_schedule_* columns, see common.GetMatchSchedule) - see
-- common.GetLeagueScheduleOverrides and engine.GenerateMatches.
CREATE TABLE IF NOT EXISTS league_schedule (
    league_id     bigint PRIMARY KEY REFERENCES leagues(league_id) ON DELETE CASCADE,
    weekdays_mask int NOT NULL,
    hour          int NOT NULL,
    minute        int NOT NULL
);

-- Name pools for two more countries the game engine can generate a world for (see
-- common.KnownCountries and handlers.AdminAddCountryHandler) - mirrors the curated ~60-name
-- style of init19_multi_country_names.sql rather than a names_es-sized dump. Unlike es/en/fr/de,
-- these are NOT inserted into `countries`/seededCountries here - they stay inactive until an
-- admin explicitly activates one via POST /v1/admin/countries.
CREATE TABLE IF NOT EXISTS names_it ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_it (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_pt ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_pt (lastname varchar(100) NULL, freq float8 NULL);

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
