-- Adds the game's first non-European countries (issue #63): Brazil, Japan, and Nigeria, one from
-- each of the three continents the issue asked for beyond Europe. Each needs its own
-- names_<code>/lastnames_<code> name-pool pair (see common.GetRandomName/nameTableSuffixByCountry)
-- - team/stadium naming data lives in Go (see common/teamnames.go, common/stadiumnames.go), not
-- the database. Countries stay inactive (not in the `countries` table) until an admin activates
-- one via POST /v1/admin/countries, same as it/pt.
--
-- Every INSERT below is guarded by "WHERE NOT EXISTS (SELECT 1 FROM <table>)" so this migration
-- stays safe to re-apply on every startup (see common.EnsureSchema/applyPendingMigrations): the
-- first run seeds an empty table, every run after that sees existing rows and no-ops.

CREATE TABLE IF NOT EXISTS names_br ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_br (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_jp ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_jp (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_ng ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ng (lastname varchar(100) NULL, freq float8 NULL);

INSERT INTO names_br ("name", freq)
SELECT * FROM (VALUES
    ('Joao',1.0),('Jose',1.0),('Pedro',1.0),('Paulo',1.0),('Carlos',1.0),('Lucas',1.0),('Gabriel',1.0),
    ('Rafael',1.0),('Bruno',1.0),('Gustavo',1.0),('Rodrigo',1.0),('Felipe',1.0),('Thiago',1.0),('Marcelo',1.0),
    ('Ricardo',1.0),('Fernando',1.0),('Eduardo',1.0),('Diego',1.0),('Vinicius',1.0),('Matheus',1.0),('Leonardo',1.0),
    ('Daniel',1.0),('Marcos',1.0),('Anderson',1.0),('Alexandre',1.0),('Fabio',1.0),('Renato',1.0),('Wesley',1.0),
    ('Everton',1.0),('Cristiano',1.0),('Douglas',1.0),('Robson',1.0),('Wagner',1.0),('Adriano',1.0),('Jefferson',1.0),
    ('Elias',1.0),('Emerson',1.0),('Junior',1.0),('Cassio',1.0),('Igor',1.0),('Vitor',1.0),('Caio',1.0),
    ('Henrique',1.0),('Guilherme',1.0),('Andre',1.0),('Luan',1.0),('Kaio',1.0),('Renan',1.0),('Willian',1.0),
    ('Ronaldo',1.0),('Roberto',1.0),('Marcio',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_br);

INSERT INTO lastnames_br (lastname, freq)
SELECT * FROM (VALUES
    ('Silva',1.0),('Santos',1.0),('Oliveira',1.0),('Souza',1.0),('Pereira',1.0),('Costa',1.0),('Rodrigues',1.0),
    ('Almeida',1.0),('Nascimento',1.0),('Lima',1.0),('Araujo',1.0),('Fernandes',1.0),('Carvalho',1.0),('Gomes',1.0),
    ('Martins',1.0),('Rocha',1.0),('Ribeiro',1.0),('Alves',1.0),('Monteiro',1.0),('Mendes',1.0),('Barros',1.0),
    ('Freitas',1.0),('Barbosa',1.0),('Pinto',1.0),('Moreira',1.0),('Cardoso',1.0),('Correia',1.0),('Teixeira',1.0),
    ('Cavalcanti',1.0),('Machado',1.0),('Ramos',1.0),('Vieira',1.0),('Dias',1.0),('Batista',1.0),('Nunes',1.0),
    ('Marques',1.0),('Castro',1.0),('Andrade',1.0),('Campos',1.0),('Cunha',1.0),('Melo',1.0),('Farias',1.0),
    ('Moura',1.0),('Azevedo',1.0),('Reis',1.0),('Sales',1.0),('Xavier',1.0),('Tavares',1.0),('Coelho',1.0),
    ('Brito',1.0),('Duarte',1.0),('Guimaraes',1.0),('Neves',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_br);

INSERT INTO names_jp ("name", freq)
SELECT * FROM (VALUES
    ('Haruto',1.0),('Yuto',1.0),('Sota',1.0),('Ryo',1.0),('Kaito',1.0),('Riku',1.0),('Sora',1.0),('Yuki',1.0),
    ('Hayato',1.0),('Taiga',1.0),('Kota',1.0),('Shota',1.0),('Kenta',1.0),('Daiki',1.0),('Kazuki',1.0),('Shun',1.0),
    ('Ren',1.0),('Tsubasa',1.0),('Naoki',1.0),('Takumi',1.0),('Yuma',1.0),('Kohei',1.0),('Ryota',1.0),('Satoshi',1.0),
    ('Yusuke',1.0),('Kenji',1.0),('Hiroshi',1.0),('Takashi',1.0),('Masato',1.0),('Akira',1.0),('Wataru',1.0),
    ('Tomoya',1.0),('Keisuke',1.0),('Genki',1.0),('Shinji',1.0),('Junya',1.0),('Hidetoshi',1.0),('Atsuto',1.0),
    ('Makoto',1.0),('Shoya',1.0),('Ritsu',1.0),('Sho',1.0),('Rikuto',1.0),('Asahi',1.0),('Minato',1.0),('Itsuki',1.0),
    ('Haru',1.0),('Sosuke',1.0),('Yamato',1.0),('Rento',1.0),('Aoi',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_jp);

INSERT INTO lastnames_jp (lastname, freq)
SELECT * FROM (VALUES
    ('Sato',1.0),('Suzuki',1.0),('Takahashi',1.0),('Tanaka',1.0),('Watanabe',1.0),('Ito',1.0),('Yamamoto',1.0),
    ('Nakamura',1.0),('Kobayashi',1.0),('Kato',1.0),('Yoshida',1.0),('Yamada',1.0),('Sasaki',1.0),('Yamaguchi',1.0),
    ('Matsumoto',1.0),('Inoue',1.0),('Kimura',1.0),('Hayashi',1.0),('Shimizu',1.0),('Yamazaki',1.0),('Mori',1.0),
    ('Abe',1.0),('Ikeda',1.0),('Hashimoto',1.0),('Yamashita',1.0),('Ishikawa',1.0),('Nakajima',1.0),('Maeda',1.0),
    ('Fujita',1.0),('Ogawa',1.0),('Goto',1.0),('Okada',1.0),('Hasegawa',1.0),('Murakami',1.0),('Kondo',1.0),
    ('Ishii',1.0),('Saito',1.0),('Sakamoto',1.0),('Endo',1.0),('Aoki',1.0),('Fujii',1.0),('Nishimura',1.0),
    ('Fukuda',1.0),('Ota',1.0),('Miura',1.0),('Fujiwara',1.0),('Okamoto',1.0),('Matsuda',1.0),('Nakagawa',1.0),
    ('Nakano',1.0),('Harada',1.0),('Ono',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_jp);

INSERT INTO names_ng ("name", freq)
SELECT * FROM (VALUES
    ('Chukwuemeka',1.0),('Emmanuel',1.0),('Ifeanyi',1.0),('Chidi',1.0),('Obinna',1.0),('Uche',1.0),('Chinedu',1.0),
    ('Ikechukwu',1.0),('Emeka',1.0),('Kelechi',1.0),('Chibuzor',1.0),('Victor',1.0),('Samuel',1.0),('Daniel',1.0),
    ('Joseph',1.0),('John',1.0),('Peter',1.0),('Paul',1.0),('Michael',1.0),('David',1.0),('Success',1.0),
    ('Godwin',1.0),('Sunday',1.0),('Friday',1.0),('Blessing',1.0),('Kingsley',1.0),('Ahmed',1.0),('Musa',1.0),
    ('Ibrahim',1.0),('Abdullahi',1.0),('Yusuf',1.0),('Aliyu',1.0),('Suleiman',1.0),('Umar',1.0),('Adamu',1.0),
    ('Bello',1.0),('Tunde',1.0),('Segun',1.0),('Femi',1.0),('Bayo',1.0),('Wale',1.0),('Yemi',1.0),('Kunle',1.0),
    ('Dele',1.0),('Gbenga',1.0),('Kayode',1.0),('Olumide',1.0),('Adewale',1.0),('Babatunde',1.0),('Adeyemi',1.0),
    ('Efe',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ng);

INSERT INTO lastnames_ng (lastname, freq)
SELECT * FROM (VALUES
    ('Okafor',1.0),('Okonkwo',1.0),('Eze',1.0),('Nwosu',1.0),('Nwachukwu',1.0),('Ibe',1.0),('Chukwu',1.0),('Obi',1.0),
    ('Anyanwu',1.0),('Uzoma',1.0),('Adebayo',1.0),('Ogundipe',1.0),('Balogun',1.0),('Afolabi',1.0),('Bakare',1.0),
    ('Ojo',1.0),('Ogunleye',1.0),('Fashola',1.0),('Okoro',1.0),('Oladipo',1.0),('Ojukwu',1.0),('Danjuma',1.0),
    ('Abubakar',1.0),('Mohammed',1.0),('Usman',1.0),('Garba',1.0),('Sani',1.0),('Yakubu',1.0),('Lawal',1.0),
    ('Shehu',1.0),('Ndubuisi',1.0),('Onyeka',1.0),('Ikenna',1.0),('Nnamdi',1.0),('Okoli',1.0),('Nnaji',1.0),
    ('Anozie',1.0),('Ugwu',1.0),('Onyekwere',1.0),('Madueke',1.0),('Iwobi',1.0),('Ndidi',1.0),('Etebo',1.0),
    ('Chukwueze',1.0),('Aina',1.0),('Ekong',1.0),('Uzoho',1.0),('Osimhen',1.0),('Simon',1.0),('Iheanacho',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ng);
