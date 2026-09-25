-- Rounds every region (see common.RegionForCountry) out to at least ten countries: 4 more for
-- Europe, 9 more for the Americas, 9 more for Asia, 9 more for Africa. Same pattern as
-- init51_more_countries.sql: each country needs its own names_<code>/lastnames_<code> pair (see
-- common.GetRandomName/nameTableSuffixByCountry) - team/stadium naming data lives in Go (see
-- common/teamnames.go, common/stadiumnames.go), not the database. Countries stay inactive (not
-- in the `countries` table) until an admin activates one via POST /v1/admin/countries.
--
-- Every INSERT below is guarded by "WHERE NOT EXISTS (SELECT 1 FROM <table>)" so this migration
-- stays safe to re-apply on every startup (see common.EnsureSchema/applyPendingMigrations): the
-- first run seeds an empty table, every run after that sees existing rows and no-ops.

CREATE TABLE IF NOT EXISTS names_nl ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_nl (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_se ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_se (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_pl ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_pl (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_gr ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_gr (lastname varchar(100) NULL, freq float8 NULL);

CREATE TABLE IF NOT EXISTS names_ar ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ar (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_mx ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_mx (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_co ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_co (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_cl ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_cl (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_pe ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_pe (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_uy ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_uy (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_us ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_us (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_ca ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ca (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_ec ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ec (lastname varchar(100) NULL, freq float8 NULL);

CREATE TABLE IF NOT EXISTS names_kr ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_kr (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_cn ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_cn (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_in ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_in (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_th ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_th (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_vn ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_vn (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_id ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_id (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_ph ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ph (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_sa ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_sa (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_ir ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ir (lastname varchar(100) NULL, freq float8 NULL);

CREATE TABLE IF NOT EXISTS names_eg ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_eg (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_gh ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_gh (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_ci ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ci (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_cm ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_cm (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_sn ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_sn (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_za ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_za (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_ma ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ma (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_dz ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_dz (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_ke ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ke (lastname varchar(100) NULL, freq float8 NULL);

-- Europe: Netherlands, Sweden, Poland, Greece

INSERT INTO names_nl ("name", freq)
SELECT * FROM (VALUES
    ('Daan',1.0),('Sem',1.0),('Lucas',1.0),('Milan',1.0),('Levi',1.0),('Luuk',1.0),('Bram',1.0),
    ('Jesse',1.0),('Finn',1.0),('Noah',1.0),('Thijs',1.0),('Sven',1.0),('Tim',1.0),('Ruben',1.0),
    ('Niels',1.0),('Joris',1.0),('Tom',1.0),('Max',1.0),('Stijn',1.0),('Cas',1.0),('Twan',1.0),
    ('Bart',1.0),('Wouter',1.0),('Jelle',1.0),('Koen',1.0),('Mees',1.0),('Pim',1.0),('Job',1.0),
    ('Dylan',1.0),('Kai',1.0),('Guus',1.0),('Boaz',1.0),('Sander',1.0),('Rick',1.0),('Wesley',1.0),
    ('Dennis',1.0),('Marco',1.0),('Erik',1.0),('Bas',1.0),('Robin',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_nl);

INSERT INTO lastnames_nl (lastname, freq)
SELECT * FROM (VALUES
    ('de Jong',1.0),('de Vries',1.0),('Jansen',1.0),('van den Berg',1.0),('van Dijk',1.0),('Bakker',1.0),
    ('Visser',1.0),('Smit',1.0),('Meijer',1.0),('Boer',1.0),('Mulder',1.0),('de Groot',1.0),
    ('Bos',1.0),('Vos',1.0),('Peters',1.0),('Hendriks',1.0),('Dekker',1.0),('Brouwer',1.0),
    ('van Leeuwen',1.0),('Dijkstra',1.0),('Smits',1.0),('de Wit',1.0),('Hoekstra',1.0),('van der Meer',1.0),
    ('de Boer',1.0),('Kok',1.0),('Postma',1.0),('Vermeulen',1.0),('van der Linden',1.0),('Bosman',1.0),
    ('van der Heijden',1.0),('Willems',1.0),('Kuiper',1.0),('van der Wal',1.0),('van Beek',1.0),
    ('van Vliet',1.0),('Verhoeven',1.0),('Verheul',1.0),('van Loon',1.0),('Timmermans',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_nl);

INSERT INTO names_se ("name", freq)
SELECT * FROM (VALUES
    ('Erik',1.0),('Lars',1.0),('Anders',1.0),('Johan',1.0),('Karl',1.0),('Nils',1.0),('Per',1.0),
    ('Gustav',1.0),('Oskar',1.0),('Emil',1.0),('Axel',1.0),('Filip',1.0),('Viktor',1.0),('Anton',1.0),
    ('Simon',1.0),('Hugo',1.0),('Elias',1.0),('Oliver',1.0),('William',1.0),('Alexander',1.0),
    ('Isak',1.0),('Leo',1.0),('Adam',1.0),('Fredrik',1.0),('Magnus',1.0),('Daniel',1.0),('Marcus',1.0),
    ('Henrik',1.0),('Mattias',1.0),('Jonas',1.0),('Tobias',1.0),('Rasmus',1.0),('Sebastian',1.0),
    ('Linus',1.0),('Martin',1.0),('Jesper',1.0),('Robin',1.0),('Andreas',1.0),('Kristian',1.0),('Niklas',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_se);

INSERT INTO lastnames_se (lastname, freq)
SELECT * FROM (VALUES
    ('Andersson',1.0),('Johansson',1.0),('Karlsson',1.0),('Nilsson',1.0),('Eriksson',1.0),('Larsson',1.0),
    ('Olsson',1.0),('Persson',1.0),('Svensson',1.0),('Gustafsson',1.0),('Pettersson',1.0),('Jonsson',1.0),
    ('Jansson',1.0),('Hansson',1.0),('Bengtsson',1.0),('Jonsson',1.0),('Lindberg',1.0),('Jakobsson',1.0),
    ('Magnusson',1.0),('Olofsson',1.0),('Lindqvist',1.0),('Lindgren',1.0),('Berg',1.0),('Bergstrom',1.0),
    ('Lundberg',1.0),('Lundqvist',1.0),('Sandberg',1.0),('Forsberg',1.0),('Nystrom',1.0),('Holmberg',1.0),
    ('Wallin',1.0),('Engstrom',1.0),('Sjoberg',1.0),('Lindstrom',1.0),('Aberg',1.0),('Ekstrom',1.0),
    ('Blom',1.0),('Nordin',1.0),('Bjork',1.0),('Soderberg',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_se);

INSERT INTO names_pl ("name", freq)
SELECT * FROM (VALUES
    ('Jakub',1.0),('Kacper',1.0),('Antoni',1.0),('Filip',1.0),('Jan',1.0),('Szymon',1.0),('Franciszek',1.0),
    ('Aleksander',1.0),('Wojciech',1.0),('Mateusz',1.0),('Michal',1.0),('Bartosz',1.0),('Piotr',1.0),
    ('Tomasz',1.0),('Krzysztof',1.0),('Marcin',1.0),('Adam',1.0),('Pawel',1.0),('Lukasz',1.0),('Rafal',1.0),
    ('Dawid',1.0),('Damian',1.0),('Patryk',1.0),('Kamil',1.0),('Sebastian',1.0),('Grzegorz',1.0),
    ('Marek',1.0),('Andrzej',1.0),('Stanislaw',1.0),('Zbigniew',1.0),('Jerzy',1.0),('Ryszard',1.0),
    ('Wladyslaw',1.0),('Henryk',1.0),('Tadeusz',1.0),('Mariusz',1.0),('Robert',1.0),('Artur',1.0),
    ('Norbert',1.0),('Ignacy',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_pl);

INSERT INTO lastnames_pl (lastname, freq)
SELECT * FROM (VALUES
    ('Nowak',1.0),('Kowalski',1.0),('Wisniewski',1.0),('Wojcik',1.0),('Kowalczyk',1.0),('Kaminski',1.0),
    ('Lewandowski',1.0),('Zielinski',1.0),('Szymanski',1.0),('Wozniak',1.0),('Dabrowski',1.0),('Kozlowski',1.0),
    ('Jankowski',1.0),('Mazur',1.0),('Wojciechowski',1.0),('Kwiatkowski',1.0),('Krawczyk',1.0),('Piotrowski',1.0),
    ('Grabowski',1.0),('Zajac',1.0),('Krol',1.0),('Wieczorek',1.0),('Jablonski',1.0),('Wrobel',1.0),
    ('Nowakowski',1.0),('Majewski',1.0),('Olszewski',1.0),('Stepien',1.0),('Malinowski',1.0),('Jaworski',1.0),
    ('Adamczyk',1.0),('Dudek',1.0),('Nowicki',1.0),('Pawlowski',1.0),('Michalski',1.0),('Krzeminski',1.0),
    ('Zalewski',1.0),('Sikora',1.0),('Walczak',1.0),('Baran',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_pl);

INSERT INTO names_gr ("name", freq)
SELECT * FROM (VALUES
    ('Georgios',1.0),('Ioannis',1.0),('Konstantinos',1.0),('Dimitrios',1.0),('Nikolaos',1.0),('Panagiotis',1.0),
    ('Vasilios',1.0),('Christos',1.0),('Michail',1.0),('Athanasios',1.0),('Andreas',1.0),('Stavros',1.0),
    ('Ilias',1.0),('Apostolos',1.0),('Grigorios',1.0),('Spyridon',1.0),('Theodoros',1.0),('Emmanouil',1.0),
    ('Alexandros',1.0),('Petros',1.0),('Charalampos',1.0),('Aristeidis',1.0),('Kyriakos',1.0),('Sotirios',1.0),
    ('Anestis',1.0),('Efstathios',1.0),('Dionysios',1.0),('Angelos',1.0),('Fotios',1.0),('Leonidas',1.0),
    ('Marios',1.0),('Nestor',1.0),('Orestis',1.0),('Panos',1.0),('Rafail',1.0),('Savvas',1.0),
    ('Timoleon',1.0),('Vlasis',1.0),('Yannis',1.0),('Zisis',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_gr);

INSERT INTO lastnames_gr (lastname, freq)
SELECT * FROM (VALUES
    ('Papadopoulos',1.0),('Papadimitriou',1.0),('Papageorgiou',1.0),('Ioannou',1.0),('Georgiou',1.0),
    ('Nikolaidis',1.0),('Dimitriou',1.0),('Vasiliou',1.0),('Konstantinou',1.0),('Christodoulou',1.0),
    ('Antoniou',1.0),('Athanasiou',1.0),('Papanikolaou',1.0),('Karagiannis',1.0),('Economou',1.0),
    ('Angelopoulos',1.0),('Makris',1.0),('Stavrou',1.0),('Pappas',1.0),('Katsaros',1.0),
    ('Triantafyllou',1.0),('Xenakis',1.0),('Stamatiou',1.0),('Panagiotopoulos',1.0),('Sideris',1.0),
    ('Kyriakou',1.0),('Michailidis',1.0),('Petrou',1.0),('Sotiriou',1.0),('Fotiadis',1.0),
    ('Grigoriou',1.0),('Andreou',1.0),('Zisimopoulos',1.0),('Vlachos',1.0),('Karydis',1.0),
    ('Mitsis',1.0),('Voulgaris',1.0),('Kontos',1.0),('Lazaridis',1.0),('Moraitis',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_gr);

-- Americas: Argentina, Mexico, Colombia, Chile, Peru, Uruguay, United States, Canada, Ecuador

INSERT INTO names_ar ("name", freq)
SELECT * FROM (VALUES
    ('Lionel',1.0),('Sergio',1.0),('Angel',1.0),('Nicolas',1.0),('Rodrigo',1.0),('Emiliano',1.0),('Lautaro',1.0),
    ('Julian',1.0),('Alejandro',1.0),('Marcos',1.0),('Facundo',1.0),('Franco',1.0),('Nahuel',1.0),
    ('Exequiel',1.0),('Enzo',1.0),('Thiago',1.0),('Santiago',1.0),('Mateo',1.0),('Bruno',1.0),('Ivan',1.0),
    ('Ezequiel',1.0),('Cristian',1.0),('German',1.0),('Leandro',1.0),('Maximiliano',1.0),('Gonzalo',1.0),
    ('Federico',1.0),('Agustin',1.0),('Joaquin',1.0),('Tomas',1.0),('Ramiro',1.0),('Ignacio',1.0),
    ('Guido',1.0),('Valentin',1.0),('Dario',1.0),('Hernan',1.0),('Matias',1.0),('Damian',1.0),
    ('Walter',1.0),('Osvaldo',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ar);

INSERT INTO lastnames_ar (lastname, freq)
SELECT * FROM (VALUES
    ('Gonzalez',1.0),('Rodriguez',1.0),('Fernandez',1.0),('Lopez',1.0),('Martinez',1.0),('Perez',1.0),
    ('Garcia',1.0),('Sanchez',1.0),('Romero',1.0),('Diaz',1.0),('Alvarez',1.0),('Torres',1.0),
    ('Ruiz',1.0),('Ramirez',1.0),('Flores',1.0),('Acosta',1.0),('Benitez',1.0),('Medina',1.0),
    ('Herrera',1.0),('Suarez',1.0),('Aguirre',1.0),('Molina',1.0),('Ortiz',1.0),('Silva',1.0),
    ('Rojas',1.0),('Nunez',1.0),('Gimenez',1.0),('Castro',1.0),('Vega',1.0),('Rios',1.0),
    ('Cabrera',1.0),('Ferreyra',1.0),('Godoy',1.0),('Ledesma',1.0),('Paez',1.0),('Quiroga',1.0),
    ('Sosa',1.0),('Villalba',1.0),('Zarate',1.0),('Bustos',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ar);

INSERT INTO names_mx ("name", freq)
SELECT * FROM (VALUES
    ('Jose',1.0),('Juan',1.0),('Carlos',1.0),('Luis',1.0),('Miguel',1.0),('Jesus',1.0),('Alejandro',1.0),
    ('Francisco',1.0),('Ricardo',1.0),('Fernando',1.0),('Roberto',1.0),('Eduardo',1.0),('Manuel',1.0),
    ('Antonio',1.0),('Sergio',1.0),('Raul',1.0),('Victor',1.0),('Alberto',1.0),('Arturo',1.0),('Ramon',1.0),
    ('Hugo',1.0),('Ivan',1.0),('Oscar',1.0),('Cesar',1.0),('Jorge',1.0),('Guillermo',1.0),('Rafael',1.0),
    ('Marco',1.0),('Enrique',1.0),('Gerardo',1.0),('Salvador',1.0),('Emilio',1.0),('Rodolfo',1.0),
    ('Adrian',1.0),('Israel',1.0),('Ulises',1.0),('Gustavo',1.0),('Erick',1.0),('Alfonso',1.0),('Rene',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_mx);

INSERT INTO lastnames_mx (lastname, freq)
SELECT * FROM (VALUES
    ('Hernandez',1.0),('Garcia',1.0),('Martinez',1.0),('Lopez',1.0),('Gonzalez',1.0),('Perez',1.0),
    ('Rodriguez',1.0),('Sanchez',1.0),('Ramirez',1.0),('Flores',1.0),('Torres',1.0),('Diaz',1.0),
    ('Vazquez',1.0),('Reyes',1.0),('Cruz',1.0),('Morales',1.0),('Ortiz',1.0),('Gutierrez',1.0),
    ('Chavez',1.0),('Ramos',1.0),('Mendoza',1.0),('Ruiz',1.0),('Aguilar',1.0),('Jimenez',1.0),
    ('Moreno',1.0),('Romero',1.0),('Alvarez',1.0),('Medina',1.0),('Castillo',1.0),('Vargas',1.0),
    ('Guzman',1.0),('Salazar',1.0),('Contreras',1.0),('Rios',1.0),('Delgado',1.0),('Fuentes',1.0),
    ('Cervantes',1.0),('Marin',1.0),('Cortez',1.0),('Espinoza',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_mx);

INSERT INTO names_co ("name", freq)
SELECT * FROM (VALUES
    ('Andres',1.0),('Carlos',1.0),('Camilo',1.0),('Santiago',1.0),('Sebastian',1.0),('Juan',1.0),('Cristian',1.0),
    ('Jhon',1.0),('Yerry',1.0),('Wilmar',1.0),('Duvan',1.0),('Miguel',1.0),('Radamel',1.0),('Luis',1.0),
    ('Jefferson',1.0),('Mateus',1.0),('Rafael',1.0),('Wilson',1.0),('Yeison',1.0),('Jorge',1.0),
    ('Fabian',1.0),('Alexis',1.0),('Cesar',1.0),('Jhonier',1.0),('Dayro',1.0),('Edwin',1.0),('Harold',1.0),
    ('Frank',1.0),('Alvaro',1.0),('Gustavo',1.0),('Nelson',1.0),('Diego',1.0),('Elkin',1.0),('Farid',1.0),
    ('Gerardo',1.0),('Kevin',1.0),('Marlon',1.0),('Steven',1.0),('Yimmi',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_co);

INSERT INTO lastnames_co (lastname, freq)
SELECT * FROM (VALUES
    ('Rodriguez',1.0),('Gomez',1.0),('Martinez',1.0),('Lopez',1.0),('Garcia',1.0),('Perez',1.0),
    ('Gonzalez',1.0),('Sanchez',1.0),('Ramirez',1.0),('Torres',1.0),('Diaz',1.0),('Vargas',1.0),
    ('Castro',1.0),('Ruiz',1.0),('Alvarez',1.0),('Romero',1.0),('Suarez',1.0),('Rojas',1.0),
    ('Moreno',1.0),('Ortiz',1.0),('Munoz',1.0),('Herrera',1.0),('Jimenez',1.0),('Ramos',1.0),
    ('Reyes',1.0),('Cardona',1.0),('Zapata',1.0),('Mejia',1.0),('Palacios',1.0),('Valencia',1.0),
    ('Arias',1.0),('Marin',1.0),('Cadavid',1.0),('Cuadrado',1.0),('Uribe',1.0),('Muriel',1.0),
    ('Guerra',1.0),('Osorio',1.0),('Pardo',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_co);

INSERT INTO names_cl ("name", freq)
SELECT * FROM (VALUES
    ('Alexis',1.0),('Arturo',1.0),('Gary',1.0),('Claudio',1.0),('Charles',1.0),('Eduardo',1.0),('Mauricio',1.0),
    ('Gonzalo',1.0),('Marcelo',1.0),('Jean',1.0),('Erick',1.0),('Ben',1.0),('Diego',1.0),('Nicolas',1.0),
    ('Bruno',1.0),('Guillermo',1.0),('Fabian',1.0),('Angelo',1.0),('Sebastian',1.0),('Junior',1.0),
    ('Igor',1.0),('Ivan',1.0),('Esteban',1.0),('Felipe',1.0),('Cristopher',1.0),('Rodrigo',1.0),
    ('Enzo',1.0),('Matias',1.0),('Pablo',1.0),('Ignacio',1.0),('Vicente',1.0),('Tomas',1.0),
    ('Cristobal',1.0),('Joaquin',1.0),('Maximiliano',1.0),('Benjamin',1.0),('Agustin',1.0),('Martin',1.0),
    ('Ricardo',1.0),('Hernan',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_cl);

INSERT INTO lastnames_cl (lastname, freq)
SELECT * FROM (VALUES
    ('Sanchez',1.0),('Sanhueza',1.0),('Aranguiz',1.0),('Valdes',1.0),('Vidal',1.0),('Isla',1.0),
    ('Medel',1.0),('Pinilla',1.0),('Diaz',1.0),('Vargas',1.0),('Silva',1.0),('Munoz',1.0),
    ('Rojas',1.0),('Fuentes',1.0),('Reyes',1.0),('Torres',1.0),('Espinoza',1.0),('Contreras',1.0),
    ('Araya',1.0),('Tapia',1.0),('Soto',1.0),('Riquelme',1.0),('Cortes',1.0),('Vera',1.0),
    ('Sepulveda',1.0),('Bravo',1.0),('Carrasco',1.0),('Fernandez',1.0),('Gutierrez',1.0),('Molina',1.0),
    ('Pizarro',1.0),('Zuniga',1.0),('Toro',1.0),('Salinas',1.0),('Orellana',1.0),('Guzman',1.0),
    ('Herrera',1.0),('Ortiz',1.0),('Rivera',1.0),('Escobar',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_cl);

INSERT INTO names_pe ("name", freq)
SELECT * FROM (VALUES
    ('Paolo',1.0),('Christian',1.0),('Jefferson',1.0),('Andre',1.0),('Yoshimar',1.0),('Edison',1.0),('Renato',1.0),
    ('Miguel',1.0),('Pedro',1.0),('Carlos',1.0),('Luis',1.0),('Gianluca',1.0),('Alexander',1.0),('Wilder',1.0),
    ('Anderson',1.0),('Aldair',1.0),('Sergio',1.0),('Raul',1.0),('Alex',1.0),('Cristian',1.0),
    ('Ray',1.0),('Beto',1.0),('Marcos',1.0),('Jose',1.0),('Franco',1.0),('Ricardo',1.0),('Alberto',1.0),
    ('Jean',1.0),('Kevin',1.0),('Nilson',1.0),('Percy',1.0),('Alexis',1.0),('Diego',1.0),('Mauricio',1.0),
    ('Cesar',1.0),('Christofer',1.0),('Renzo',1.0),('Piero',1.0),('Junior',1.0),('Fernando',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_pe);

INSERT INTO lastnames_pe (lastname, freq)
SELECT * FROM (VALUES
    ('Guerrero',1.0),('Farfan',1.0),('Vargas',1.0),('Cueva',1.0),('Flores',1.0),('Advincula',1.0),
    ('Trauco',1.0),('Zambrano',1.0),('Carrillo',1.0),('Yotun',1.0),('Tapia',1.0),('Lapadula',1.0),
    ('Ramos',1.0),('Lopez',1.0),('Rodriguez',1.0),('Chavez',1.0),('Rojas',1.0),('Castillo',1.0),
    ('Reyes',1.0),('Torres',1.0),('Salazar',1.0),('Quispe',1.0),('Mamani',1.0),('Huaman',1.0),
    ('Condori',1.0),('Aguilar',1.0),('Cardenas',1.0),('Mendoza',1.0),('Palomino',1.0),('Zevallos',1.0),
    ('Ponce',1.0),('Delgado',1.0),('Villanueva',1.0),('Nunez',1.0),('Cruzado',1.0),('Guevara',1.0),
    ('Solano',1.0),('Requena',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_pe);

INSERT INTO names_uy ("name", freq)
SELECT * FROM (VALUES
    ('Luis',1.0),('Edinson',1.0),('Diego',1.0),('Federico',1.0),('Jose',1.0),('Nahitan',1.0),('Rodrigo',1.0),
    ('Matias',1.0),('Fede',1.0),('Ronald',1.0),('Martin',1.0),('Giorgian',1.0),('Lucas',1.0),('Sebastian',1.0),
    ('Gaston',1.0),('Cristian',1.0),('Bruno',1.0),('Nicolas',1.0),('Maximiliano',1.0),('Guillermo',1.0),
    ('Gonzalo',1.0),('Facundo',1.0),('Agustin',1.0),('Sergio',1.0),('Alvaro',1.0),('Mathias',1.0),
    ('Fabian',1.0),('Jonathan',1.0),('Emiliano',1.0),('Franco',1.0),('Damian',1.0),('Ignacio',1.0),
    ('Marcelo',1.0),('Walter',1.0),('Andres',1.0),('Pablo',1.0),('Fernando',1.0),('Leandro',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_uy);

INSERT INTO lastnames_uy (lastname, freq)
SELECT * FROM (VALUES
    ('Suarez',1.0),('Cavani',1.0),('Godin',1.0),('Bentancur',1.0),('Valverde',1.0),('Araujo',1.0),
    ('Torreira',1.0),('Vecino',1.0),('Nunez',1.0),('Rodriguez',1.0),('Pereira',1.0),('Coates',1.0),
    ('Gimenez',1.0),('Cristoforo',1.0),('Rolan',1.0),('Cabrera',1.0),('Vina',1.0),('Olivera',1.0),
    ('Pellistri',1.0),('Ugarte',1.0),('Ramirez',1.0),('Silva',1.0),('Fernandez',1.0),('Gonzalez',1.0),
    ('Perez',1.0),('Lopez',1.0),('Martinez',1.0),('Alonso',1.0),('Sosa',1.0),('Machado',1.0),
    ('Rivas',1.0),('Techera',1.0),('Corujo',1.0),('Guzman',1.0),('Cardozo',1.0),('Britos',1.0),
    ('Amaral',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_uy);

INSERT INTO names_us ("name", freq)
SELECT * FROM (VALUES
    ('Christian',1.0),('Weston',1.0),('Tyler',1.0),('Gio',1.0),('Sergino',1.0),('Tim',1.0),('Brenden',1.0),
    ('Ricardo',1.0),('Josh',1.0),('Antonee',1.0),('Walker',1.0),('Matt',1.0),('Zack',1.0),('Yunus',1.0),
    ('Paul',1.0),('Landon',1.0),('Clint',1.0),('Michael',1.0),('Jozy',1.0),('Kellyn',1.0),
    ('Cole',1.0),('Aaron',1.0),('Brandon',1.0),('Cameron',1.0),('Erik',1.0),('Jordan',1.0),('Reggie',1.0),
    ('Jesse',1.0),('John',1.0),('Kyle',1.0),('Nick',1.0),('Ryan',1.0),('Scott',1.0),('Steve',1.0),
    ('Brian',1.0),('Frankie',1.0),('Sam',1.0),('Ethan',1.0),('Justin',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_us);

INSERT INTO lastnames_us (lastname, freq)
SELECT * FROM (VALUES
    ('Pulisic',1.0),('McKennie',1.0),('Reyna',1.0),('Adams',1.0),('Dest',1.0),('Ream',1.0),
    ('Aaronson',1.0),('Turner',1.0),('Robinson',1.0),('Sargent',1.0),('Musah',1.0),('Zimmerman',1.0),
    ('Scally',1.0),('Weah',1.0),('Yedlin',1.0),('Bradley',1.0),('Howard',1.0),('Dempsey',1.0),
    ('Donovan',1.0),('Altidore',1.0),('Wood',1.0),('Zardes',1.0),('Morris',1.0),('Acosta',1.0),
    ('Roldan',1.0),('Arriola',1.0),('Boyd',1.0),('Lletget',1.0),('Long',1.0),('Miazga',1.0),
    ('Ledezma',1.0),('Pepi',1.0),('Balogun',1.0),('Tillman',1.0),('Richards',1.0),('Horvath',1.0),
    ('Steffen',1.0),('Vazquez',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_us);

INSERT INTO names_ca ("name", freq)
SELECT * FROM (VALUES
    ('Alphonso',1.0),('Jonathan',1.0),('Cyle',1.0),('Tajon',1.0),('Stephen',1.0),('Junior',1.0),('Sam',1.0),
    ('Jamal',1.0),('Alistair',1.0),('Milan',1.0),('Kamal',1.0),('Liam',1.0),('Doneil',1.0),('Jayden',1.0),
    ('Richie',1.0),('David',1.0),('Atiba',1.0),('Scott',1.0),('Ike',1.0),('Dejan',1.0),
    ('Maxime',1.0),('Ismael',1.0),('Lucas',1.0),('Samuel',1.0),('Jacen',1.0),('Kadin',1.0),('Zachary',1.0),
    ('Marco',1.0),('Steven',1.0),('Russell',1.0),('Dominick',1.0),('Marcus',1.0),('Derek',1.0),
    ('Adam',1.0),('Terry',1.0),('Dean',1.0),('Julian',1.0),('Charles',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ca);

INSERT INTO lastnames_ca (lastname, freq)
SELECT * FROM (VALUES
    ('Davies',1.0),('David',1.0),('Larin',1.0),('Buchanan',1.0),('Eustaquio',1.0),('Osorio',1.0),
    ('Adekugbe',1.0),('Hutchinson',1.0),('Johnston',1.0),('Kone',1.0),('Miller',1.0),('Cornelius',1.0),
    ('Millar',1.0),('Laryea',1.0),('Cavallini',1.0),('Piette',1.0),('Wotherspoon',1.0),('Henry',1.0),
    ('Hoilett',1.0),('Bustos',1.0),('Waterman',1.0),('James',1.0),('Vitoria',1.0),
    ('Akindele',1.0),('Ledgerwood',1.0),('Borjan',1.0),('Crepeau',1.0),('Doyle',1.0),
    ('Priso',1.0),('Roberts',1.0),('Petrasso',1.0),('Kaye',1.0),('Shaffelburg',1.0),
    ('Brault-Guillard',1.0),('Jackson',1.0),('Baldisimo',1.0),('Metcalfe',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ca);

INSERT INTO names_ec ("name", freq)
SELECT * FROM (VALUES
    ('Enner',1.0),('Michael',1.0),('Moises',1.0),('Byron',1.0),('Pervis',1.0),('Piero',1.0),('Angelo',1.0),
    ('Gonzalo',1.0),('Alexander',1.0),('Felix',1.0),('Jefferson',1.0),('Robert',1.0),('Xavier',1.0),
    ('Antonio',1.0),('Jhegson',1.0),('Renato',1.0),('Jose',1.0),('Fernando',1.0),('Cristian',1.0),
    ('Pedro',1.0),('Diego',1.0),('Carlos',1.0),('Miguel',1.0),('Alan',1.0),('Adrian',1.0),('Nilson',1.0),
    ('Bryan',1.0),('Kevin',1.0),('Ronny',1.0),('Cristhian',1.0),('Rodrigo',1.0),('Damian',1.0),
    ('Israel',1.0),('Jean',1.0),('Ricardo',1.0),('Junior',1.0),('Marlon',1.0),('Oscar',1.0),('Wellington',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ec);

INSERT INTO lastnames_ec (lastname, freq)
SELECT * FROM (VALUES
    ('Valencia',1.0),('Caicedo',1.0),('Preciado',1.0),('Estrada',1.0),('Mendez',1.0),('Ibarra',1.0),
    ('Plata',1.0),('Hincapie',1.0),('Pacho',1.0),('Sarmiento',1.0),('Torres',1.0),('Arroyo',1.0),
    ('Cifuentes',1.0),('Mena',1.0),('Angulo',1.0),('Quinonez',1.0),('Rodriguez',1.0),('Perlaza',1.0),
    ('Gruezo',1.0),('Franco',1.0),('Bone',1.0),('Cazares',1.0),('Quinteros',1.0),('Chala',1.0),
    ('Castillo',1.0),('Vera',1.0),('Zambrano',1.0),('Delgado',1.0),('Macias',1.0),('Segura',1.0),
    ('Lopez',1.0),('Tenorio',1.0),('Reasco',1.0),('Cortez',1.0),('Minda',1.0),('Congo',1.0),
    ('Ordonez',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ec);

-- Asia: South Korea, China, India, Thailand, Vietnam, Indonesia, Philippines, Saudi Arabia, Iran

INSERT INTO names_kr ("name", freq)
SELECT * FROM (VALUES
    ('Min-jae',1.0),('Heung-min',1.0),('Kang-in',1.0),('Woo-yeong',1.0),('Hee-chan',1.0),('Jae-sung',1.0),
    ('In-beom',1.0),('Seung-ho',1.0),('Young-gwon',1.0),('Dong-gyeong',1.0),('Hyun-woo',1.0),('Kyung-won',1.0),
    ('Jung-hoon',1.0),('Chang-hoon',1.0),('Do-hyun',1.0),('Ui-jo',1.0),('Jong-woo',1.0),('Jin-su',1.0),
    ('Jae-hyeok',1.0),('Sung-yueng',1.0),('Seon-min',1.0),('Chan-ho',1.0),('Tae-hwan',1.0),
    ('Chung-yong',1.0),('Hyung-jin',1.0),('Jung-woo',1.0),('Yun-do',1.0),('Sung-hoon',1.0),('Ye-chan',1.0),
    ('Woo-hyun',1.0),('Kyu-sung',1.0),('Se-jong',1.0),('Do-yeon',1.0),('Jae-won',1.0),('Seong-yoon',1.0),
    ('Han-bin',1.0),('Yong-woo',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_kr);

INSERT INTO lastnames_kr (lastname, freq)
SELECT * FROM (VALUES
    ('Kim',1.0),('Lee',1.0),('Park',1.0),('Choi',1.0),('Jung',1.0),('Kang',1.0),('Cho',1.0),
    ('Yoon',1.0),('Jang',1.0),('Lim',1.0),('Han',1.0),('Oh',1.0),('Seo',1.0),('Shin',1.0),
    ('Kwon',1.0),('Hwang',1.0),('Ahn',1.0),('Song',1.0),('Yoo',1.0),('Hong',1.0),
    ('Ko',1.0),('Moon',1.0),('Yang',1.0),('Bae',1.0),('Baek',1.0),('Nam',1.0),('Ryu',1.0),
    ('Ha',1.0),('Gu',1.0),('Ok',1.0),('Chun',1.0),('Ju',1.0),('Do',1.0),('Byun',1.0),
    ('Jeon',1.0),('Gong',1.0),('Rho',1.0),('Sim',1.0),('Woo',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_kr);

INSERT INTO names_cn ("name", freq)
SELECT * FROM (VALUES
    ('Wei',1.0),('Jun',1.0),('Hao',1.0),('Yang',1.0),('Lei',1.0),('Peng',1.0),('Bo',1.0),('Chao',1.0),
    ('Fei',1.0),('Gang',1.0),('Tao',1.0),('Kai',1.0),('Cheng',1.0),('Hui',1.0),('Jian',1.0),('Long',1.0),
    ('Ming',1.0),('Qiang',1.0),('Wen',1.0),('Xin',1.0),('Yong',1.0),('Zhi',1.0),('Ping',1.0),('Feng',1.0),
    ('Hong',1.0),('Guo',1.0),('Bin',1.0),('Da',1.0),('Rui',1.0),('Xiang',1.0),('Zheng',1.0),('Jia',1.0),
    ('Song',1.0),('Liang',1.0),('Bao',1.0),('Fan',1.0),('Han',1.0),('Xun',1.0),('Zhen',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_cn);

INSERT INTO lastnames_cn (lastname, freq)
SELECT * FROM (VALUES
    ('Wang',1.0),('Li',1.0),('Zhang',1.0),('Liu',1.0),('Chen',1.0),('Yang',1.0),('Huang',1.0),('Zhao',1.0),
    ('Wu',1.0),('Zhou',1.0),('Xu',1.0),('Sun',1.0),('Ma',1.0),('Zhu',1.0),('Hu',1.0),('Guo',1.0),
    ('He',1.0),('Lin',1.0),('Gao',1.0),('Luo',1.0),('Zheng',1.0),('Liang',1.0),('Xie',1.0),('Song',1.0),
    ('Tang',1.0),('Han',1.0),('Feng',1.0),('Deng',1.0),('Cao',1.0),('Peng',1.0),('Zeng',1.0),('Xiao',1.0),
    ('Tian',1.0),('Dong',1.0),('Yuan',1.0),('Pan',1.0),('Cai',1.0),('Jiang',1.0),('Yu',1.0),('Du',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_cn);

INSERT INTO names_in ("name", freq)
SELECT * FROM (VALUES
    ('Sunil',1.0),('Sandesh',1.0),('Gurpreet',1.0),('Anirudh',1.0),('Rahul',1.0),('Sahal',1.0),('Manvir',1.0),
    ('Udanta',1.0),('Liston',1.0),('Pritam',1.0),('Adil',1.0),('Amarjit',1.0),('Anwar',1.0),('Ashique',1.0),
    ('Brandon',1.0),('Deepak',1.0),('Jeakson',1.0),('Nikhil',1.0),
    ('Pronay',1.0),('Rowllin',1.0),('Suresh',1.0),('Vishal',1.0),('Yumnam',1.0),('Akash',1.0),('Bikash',1.0),
    ('Farukh',1.0),('Gurmeet',1.0),('Harmanjot',1.0),('Ishan',1.0),('Jitendra',1.0),('Karan',1.0),
    ('Milan',1.0),('Naorem',1.0),('Omkar',1.0),('Prabir',1.0),('Rajat',1.0),('Simranjeet',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_in);

INSERT INTO lastnames_in (lastname, freq)
SELECT * FROM (VALUES
    ('Chhetri',1.0),('Singh',1.0),('Sandhu',1.0),('Kumar',1.0),('Yadav',1.0),('Sharma',1.0),('Verma',1.0),
    ('Gupta',1.0),('Mehta',1.0),('Kapoor',1.0),('Chauhan',1.0),('Rana',1.0),('Thapa',1.0),('Gowda',1.0),
    ('Nair',1.0),('Menon',1.0),('Iyer',1.0),('Reddy',1.0),('Rao',1.0),('Pillai',1.0),
    ('Bora',1.0),('Khan',1.0),('Ali',1.0),('Ansari',1.0),('Shaikh',1.0),('Chowdhury',1.0),('Bose',1.0),
    ('Dutta',1.0),('Sengupta',1.0),('Mukherjee',1.0),('Banerjee',1.0),('Ghosh',1.0),('Das',1.0),
    ('Roy',1.0),('Sarkar',1.0),('Bhatt',1.0),('Joshi',1.0),('Malhotra',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_in);

INSERT INTO names_th ("name", freq)
SELECT * FROM (VALUES
    ('Chanathip',1.0),('Theerathon',1.0),('Teerasil',1.0),('Adisak',1.0),('Sarach',1.0),('Kawin',1.0),
    ('Pansa',1.0),('Supachok',1.0),('Manuel',1.0),('Elias',1.0),('Bordin',1.0),('Peeradol',1.0),
    ('Nitipong',1.0),('Sasalak',1.0),('Weerathep',1.0),('Thitipan',1.0),('Korrakot',1.0),('Ekanit',1.0),
    ('Siwakorn',1.0),('Anon',1.0),('Ben',1.0),('Chananan',1.0),('Ekkachai',1.0),
    ('Firdaus',1.0),('Godpop',1.0),('Issawa',1.0),('Jetsada',1.0),('Kritsada',1.0),
    ('Mongkol',1.0),('Narit',1.0),('Peerapat',1.0),('Rangsan',1.0),('Sittichok',1.0),('Tanaboon',1.0),
    ('Watcharapong',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_th);

INSERT INTO lastnames_th (lastname, freq)
SELECT * FROM (VALUES
    ('Songkrasin',1.0),('Chaikwan',1.0),('Nualsri',1.0),('Yoonan',1.0),('Dangda',1.0),('Pokklaw',1.0),
    ('Chotnamniyom',1.0),('Anusorn',1.0),('Panyachalerm',1.0),('Suthirat',1.0),('Aiyarawit',1.0),
    ('Kesorn',1.0),('Preedee',1.0),('Sompong',1.0),('Rattanakorn',1.0),('Thawornsak',1.0),
    ('Wongsuwan',1.0),('Wichaya',1.0),('Sakulwiwatana',1.0),('Nualpakdee',1.0),
    ('Boonlorm',1.0),('Rattanawan',1.0),('Thongchai',1.0),('Sirisophon',1.0),('Panpim',1.0),
    ('Kittikun',1.0),('Sombat',1.0),('Chartpanit',1.0),('Wongchan',1.0),('Suksawat',1.0),
    ('Boonprasert',1.0),('Thanasan',1.0),('Manirot',1.0),('Pattamakiat',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_th);

INSERT INTO names_vn ("name", freq)
SELECT * FROM (VALUES
    ('Van',1.0),('Cong',1.0),('Quang',1.0),('Duc',1.0),('Minh',1.0),('Hoang',1.0),('Anh',1.0),('Tuan',1.0),
    ('Thanh',1.0),('Hung',1.0),('Duy',1.0),('Nam',1.0),('Trung',1.0),('Phuc',1.0),('Long',1.0),('Khanh',1.0),
    ('Bao',1.0),('Dat',1.0),('Hai',1.0),('Kien',1.0),('Loc',1.0),('Nghia',1.0),('Phong',1.0),('Quoc',1.0),
    ('Son',1.0),('Thai',1.0),('Truong',1.0),('Vu',1.0),('Xuan',1.0),('Binh',1.0),('Chinh',1.0),('Dinh',1.0),
    ('Giang',1.0),('Hieu',1.0),('Khai',1.0),('Loi',1.0),('Manh',1.0),('Ninh',1.0),('Phat',1.0),('Sang',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_vn);

INSERT INTO lastnames_vn (lastname, freq)
SELECT * FROM (VALUES
    ('Nguyen',1.0),('Tran',1.0),('Le',1.0),('Pham',1.0),('Hoang',1.0),('Huynh',1.0),('Phan',1.0),
    ('Vu',1.0),('Vo',1.0),('Dang',1.0),('Bui',1.0),('Do',1.0),('Ho',1.0),('Ngo',1.0),
    ('Duong',1.0),('Ly',1.0),('Trinh',1.0),('Dinh',1.0),('Ta',1.0),('Truong',1.0),
    ('Dao',1.0),('Mai',1.0),('Vuong',1.0),('Luu',1.0),('Cao',1.0),('Chu',1.0),('Dam',1.0),
    ('Doan',1.0),('Han',1.0),('Kieu',1.0),('Luong',1.0),('Ma',1.0),('Nong',1.0),('Quach',1.0),
    ('Quan',1.0),('Thai',1.0),('Thao',1.0),('Trang',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_vn);

INSERT INTO names_id ("name", freq)
SELECT * FROM (VALUES
    ('Egy',1.0),('Witan',1.0),('Asnawi',1.0),('Rachmat',1.0),('Marc',1.0),('Ricky',1.0),('Ezra',1.0),
    ('Rizky',1.0),('Pratama',1.0),('Bagas',1.0),('Ronaldo',1.0),('Muhammad',1.0),('Kevin',1.0),('Diego',1.0),
    ('Beckham',1.0),('Fajar',1.0),('Elkan',1.0),('Yakob',1.0),('Irfan',1.0),('Feby',1.0),
    ('Saddil',1.0),('Osvaldo',1.0),('Andy',1.0),('Iwan',1.0),('Bayu',1.0),('Yudha',1.0),('Dedik',1.0),
    ('Wawan',1.0),('Zulfiandi',1.0),('Sandy',1.0),('Fachruddin',1.0),('Rizki',1.0),
    ('Braif',1.0),('Gian',1.0),('Kadek',1.0),('Rahmat',1.0),('Arya',1.0),('Fikri',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_id);

INSERT INTO lastnames_id (lastname, freq)
SELECT * FROM (VALUES
    ('Maulana',1.0),('Wibowo',1.0),('Setiawan',1.0),('Hakim',1.0),('Ramadhan',1.0),('Susanto',1.0),
    ('Utomo',1.0),('Santoso',1.0),('Kurniawan',1.0),('Saputra',1.0),('Nugroho',1.0),('Pratama',1.0),
    ('Widodo',1.0),('Suryanto',1.0),('Gunawan',1.0),('Wijaya',1.0),('Firmansyah',1.0),('Yulianto',1.0),
    ('Iskandar',1.0),('Prasetyo',1.0),('Handoko',1.0),('Purnomo',1.0),('Kusuma',1.0),
    ('Halim',1.0),('Effendi',1.0),('Siregar',1.0),('Sinaga',1.0),('Hutagalung',1.0),('Simanjuntak',1.0),
    ('Panjaitan',1.0),('Nasution',1.0),('Harahap',1.0),('Situmorang',1.0),('Marpaung',1.0),
    ('Sitorus',1.0),('Tampubolon',1.0),('Pardede',1.0),('Silalahi',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_id);

INSERT INTO names_ph ("name", freq)
SELECT * FROM (VALUES
    ('Stephan',1.0),('Patrick',1.0),('Angel',1.0),('Iain',1.0),('Amani',1.0),('Justin',1.0),('Manny',1.0),
    ('Simone',1.0),('Kevin',1.0),('Daisuke',1.0),('Mark',1.0),('Jarvey',1.0),('Sandro',1.0),
    ('Miguel',1.0),('Michael',1.0),('Joshua',1.0),('Zico',1.0),('Jefferson',1.0),
    ('Anton',1.0),('Marius',1.0),('Alfred',1.0),('Curt',1.0),('Ray',1.0),('Karl',1.0),('Chad',1.0),
    ('Dylan',1.0),('Julian',1.0),('Nathaniel',1.0),('Christian',1.0),('Gerald',1.0),('Randolf',1.0),
    ('Kaya',1.0),('Robert',1.0),('Ernie',1.0),('Ray',1.0),('Jake',1.0),('Arnold',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ph);

INSERT INTO lastnames_ph (lastname, freq)
SELECT * FROM (VALUES
    ('Schrock',1.0),('Reichelt',1.0),('Gonzales',1.0),('Fletcher',1.0),('Guirado',1.0),('Baas',1.0),
    ('Aguinaldo',1.0),('Ott',1.0),('Younghusband',1.0),('Amida',1.0),('Suzuki',1.0),('Hartmann',1.0),
    ('Cabigting',1.0),('Porteria',1.0),('Mora',1.0),('Musico',1.0),('Malicse',1.0),
    ('Etheridge',1.0),('Denmark',1.0),('Yatar',1.0),('Ike',1.0),('Pasilan',1.0),('Manalang',1.0),
    ('Del Rosario',1.0),('Sanvictores',1.0),('Alcantara',1.0),('Miller',1.0),
    ('Villahermosa',1.0),('Dennehy',1.0),('Almira',1.0),('Alegre',1.0),('Marasigan',1.0),
    ('Palla',1.0),('Barbon',1.0),('McDaniel',1.0),('Reyes',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ph);

INSERT INTO names_sa ("name", freq)
SELECT * FROM (VALUES
    ('Salem',1.0),('Salman',1.0),('Firas',1.0),('Fahad',1.0),('Abdullah',1.0),('Yasser',1.0),('Nawaf',1.0),
    ('Saud',1.0),('Mohammed',1.0),('Yahya',1.0),('Sultan',1.0),('Nasser',1.0),('Khalid',1.0),
    ('Abdulrahman',1.0),('Turki',1.0),('Faisal',1.0),('Hattan',1.0),('Majed',1.0),('Ziyad',1.0),
    ('Hamzah',1.0),('Sami',1.0),('Waleed',1.0),('Basil',1.0),('Raed',1.0),('Naif',1.0),('Talal',1.0),
    ('Bandar',1.0),('Rakan',1.0),('Musab',1.0),('Ayman',1.0),('Fawaz',1.0),('Hesham',1.0),
    ('Marzouq',1.0),('Osama',1.0),('Qusai',1.0),('Rayan',1.0),('Suhail',1.0),('Thamer',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_sa);

INSERT INTO lastnames_sa (lastname, freq)
SELECT * FROM (VALUES
    ('Al-Dawsari',1.0),('Al-Faraj',1.0),('Al-Shehri',1.0),('Al-Owais',1.0),('Al-Buraikan',1.0),
    ('Al-Bulaihi',1.0),('Al-Amri',1.0),('Al-Ghannam',1.0),('Al-Malki',1.0),('Al-Hamdan',1.0),
    ('Al-Sahafi',1.0),('Al-Yami',1.0),('Al-Harbi',1.0),('Al-Otaibi',1.0),('Al-Ghamdi',1.0),
    ('Al-Zahrani',1.0),('Al-Qahtani',1.0),('Al-Anazi',1.0),('Al-Enezi',1.0),('Al-Juhani',1.0),
    ('Al-Balawi',1.0),('Al-Mutairi',1.0),('Al-Ahmadi',1.0),('Al-Khaibari',1.0),('Al-Rashidi',1.0),
    ('Al-Subaie',1.0),('Al-Yousef',1.0),('Al-Nasser',1.0),('Al-Fahad',1.0),('Al-Turki',1.0),
    ('Al-Salem',1.0),('Al-Mansour',1.0),('Al-Ammar',1.0),('Al-Jaber',1.0),('Al-Suwailem',1.0),
    ('Al-Hazmi',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_sa);

INSERT INTO names_ir ("name", freq)
SELECT * FROM (VALUES
    ('Mehdi',1.0),('Sardar',1.0),('Alireza',1.0),('Karim',1.0),('Vahid',1.0),('Saman',1.0),('Ehsan',1.0),
    ('Ashkan',1.0),('Ramin',1.0),('Sepehr',1.0),('Milad',1.0),('Omid',1.0),('Reza',1.0),('Ali',1.0),
    ('Hossein',1.0),('Mohammad',1.0),('Amir',1.0),('Payam',1.0),('Morteza',1.0),('Shoja',1.0),
    ('Farshad',1.0),('Danial',1.0),('Roozbeh',1.0),('Saeid',1.0),('Hamid',1.0),('Kaveh',1.0),
    ('Bahram',1.0),('Arash',1.0),('Babak',1.0),('Cyrus',1.0),('Dariush',1.0),('Farhad',1.0),
    ('Hooman',1.0),('Iman',1.0),('Javad',1.0),('Kian',1.0),('Nima',1.0),('Peyman',1.0),('Rostam',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ir);

INSERT INTO lastnames_ir (lastname, freq)
SELECT * FROM (VALUES
    ('Taremi',1.0),('Azmoun',1.0),('Jahanbakhsh',1.0),('Ansarifard',1.0),('Hosseini',1.0),
    ('Pouraliganji',1.0),('Moharrami',1.0),('Shojaei',1.0),('Ezatolahi',1.0),('Karimi',1.0),
    ('Rezaeian',1.0),('Ghoddos',1.0),('Amiri',1.0),('Cheshmi',1.0),('Hajsafi',1.0),
    ('Nourollahi',1.0),('Rafiei',1.0),('Sadeghi',1.0),('Hashemian',1.0),
    ('Daei',1.0),('Bagheri',1.0),('Khalatbari',1.0),('Mansourian',1.0),('Nekounam',1.0),
    ('Teymourian',1.0),('Kalantari',1.0),('Madanchi',1.0),('Motahari',1.0),
    ('Naderi',1.0),('Piroozi',1.0),('Rahnama',1.0),('Sabet',1.0),('Taghavi',1.0),('Vali',1.0),
    ('Zamani',1.0),('Askari',1.0),('Bayat',1.0),('Farahani',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ir);

-- Africa: Egypt, Ghana, Ivory Coast, Cameroon, Senegal, South Africa, Morocco, Algeria, Kenya

INSERT INTO names_eg ("name", freq)
SELECT * FROM (VALUES
    ('Mohamed',1.0),('Mahmoud',1.0),('Ahmed',1.0),('Omar',1.0),('Amr',1.0),('Hossam',1.0),('Karim',1.0),
    ('Sherif',1.0),('Ramadan',1.0),('Marwan',1.0),('Emam',1.0),('Ayman',1.0),('Tarek',1.0),
    ('Hassan',1.0),('Hany',1.0),('Nabil',1.0),('Sameh',1.0),('Osama',1.0),('Wael',1.0),('Abdallah',1.0),
    ('Yasser',1.0),('Ibrahim',1.0),('Khaled',1.0),('Islam',1.0),('Mostafa',1.0),('Akram',1.0),
    ('Bassem',1.0),('Fathy',1.0),('Gamal',1.0),('Hazem',1.0),('Ihab',1.0),('Jamal',1.0),
    ('Kareem',1.0),('Loay',1.0),('Magdy',1.0),('Nasser',1.0),('Rami',1.0),('Salah',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_eg);

INSERT INTO lastnames_eg (lastname, freq)
SELECT * FROM (VALUES
    ('El-Nenny',1.0),('Elneny',1.0),('Elmohamady',1.0),('Hegazi',1.0),('Fathy',1.0),
    ('El-Shenawy',1.0),('Marmoush',1.0),('Ashour',1.0),('Kahraba',1.0),('Ramadan',1.0),
    ('Soliman',1.0),('Salah',1.0),('Kamal',1.0),('Abdelmonem',1.0),('Hamdy',1.0),('Sabry',1.0),
    ('Farag',1.0),('Mahmoud',1.0),('Ibrahim',1.0),('Kamel',1.0),('Fouad',1.0),
    ('Rashad',1.0),('Shawky',1.0),('Wahba',1.0),('Yassin',1.0),('Amer',1.0),('Badr',1.0),
    ('Dessouki',1.0),('Emara',1.0),('Farid',1.0),('Ghali',1.0),('Hosny',1.0),('Kassem',1.0),
    ('Mansour',1.0),('Naguib',1.0),('Rifai',1.0),('Sultan',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_eg);

INSERT INTO names_gh ("name", freq)
SELECT * FROM (VALUES
    ('Thomas',1.0),('Andre',1.0),('Jordan',1.0),('Mohammed',1.0),('Iddrisu',1.0),('Baba',1.0),
    ('Osman',1.0),('Alexander',1.0),('Daniel',1.0),('Joseph',1.0),('Jonathan',1.0),('Richmond',1.0),
    ('Christopher',1.0),('Emmanuel',1.0),('Samuel',1.0),('Benjamin',1.0),('Isaac',1.0),('Prince',1.0),
    ('Kwame',1.0),('Kofi',1.0),('Kwabena',1.0),('Yaw',1.0),('Kwaku',1.0),('Nana',1.0),
    ('Ebenezer',1.0),('Fiifi',1.0),('Gideon',1.0),('Ishmael',1.0),('Justice',1.0),('Lawrence',1.0),
    ('Michael',1.0),('Nathaniel',1.0),('Obed',1.0),('Patrick',1.0),('Quincy',1.0),('Solomon',1.0),
    ('Vincent',1.0),('Wisdom',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_gh);

INSERT INTO lastnames_gh (lastname, freq)
SELECT * FROM (VALUES
    ('Partey',1.0),('Ayew',1.0),('Amartey',1.0),('Salisu',1.0),('Kudus',1.0),('Baba',1.0),
    ('Semenyo',1.0),('Sarkodie',1.0),('Djiku',1.0),('Mensah',1.0),('Owusu',1.0),('Boateng',1.0),
    ('Asante',1.0),('Appiah',1.0),('Adjei',1.0),('Amoah',1.0),('Osei',1.0),('Yeboah',1.0),
    ('Frimpong',1.0),('Acheampong',1.0),('Antwi',1.0),('Bediako',1.0),('Danso',1.0),('Essien',1.0),
    ('Gyan',1.0),('Gyasi',1.0),('Kwarteng',1.0),('Larbi',1.0),('Ntiamoah',1.0),('Opoku',1.0),
    ('Quansah',1.0),('Sarpong',1.0),('Tetteh',1.0),('Twumasi',1.0),('Wiredu',1.0),('Yankey',1.0),
    ('Addo',1.0),('Baah',1.0),('Dapaah',1.0),('Frempong',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_gh);

INSERT INTO names_ci ("name", freq)
SELECT * FROM (VALUES
    ('Didier',1.0),('Yaya',1.0),('Kolo',1.0),('Wilfried',1.0),('Serge',1.0),('Franck',1.0),('Nicolas',1.0),
    ('Ibrahim',1.0),('Eric',1.0),('Odilon',1.0),('Simon',1.0),('Cheick',1.0),('Serey',1.0),
    ('Ghislain',1.0),('Amad',1.0),('Sebastien',1.0),('Willy',1.0),('Christian',1.0),
    ('Boubacar',1.0),('Ismael',1.0),('Souleymane',1.0),('Karim',1.0),('Mory',1.0),('Adama',1.0),
    ('Bafetimbi',1.0),('Emmanuel',1.0),('Gervinho',1.0),('Guy',1.0),('Hamed',1.0),('Jonathan',1.0),
    ('Lacina',1.0),('Mamadou',1.0),('Ousmane',1.0),('Salomon',1.0),('Siaka',1.0),('Tiemoue',1.0),('Zie',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ci);

INSERT INTO lastnames_ci (lastname, freq)
SELECT * FROM (VALUES
    ('Drogba',1.0),('Toure',1.0),('Zaha',1.0),('Gradel',1.0),('Aurier',1.0),('Kessie',1.0),
    ('Fofana',1.0),('Aholou',1.0),('Sangare',1.0),('Konan',1.0),('Kone',1.0),('Bailly',1.0),
    ('Traore',1.0),('Coulibaly',1.0),('Diomande',1.0),('Doumbia',1.0),('Sanogo',1.0),('Diakite',1.0),
    ('Kouassi',1.0),('Yao',1.0),('Kouame',1.0),('Ouattara',1.0),('Bamba',1.0),('Diarra',1.0),
    ('Kobenan',1.0),('Meite',1.0),('Pokou',1.0),('Silue',1.0),('Tanoh',1.0),
    ('Yapi',1.0),('Zadi',1.0),('Angoua',1.0),('Bakayoko',1.0),('Diabate',1.0),('Kanga',1.0),('Yeo',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ci);

INSERT INTO names_cm ("name", freq)
SELECT * FROM (VALUES
    ('Samuel',1.0),('Andre',1.0),('Vincent',1.0),('Eric',1.0),('Joel',1.0),('Nicolas',1.0),('Karl',1.0),
    ('Christian',1.0),('Jean',1.0),('Bryan',1.0),('Martin',1.0),('Ambroise',1.0),('Yves',1.0),
    ('Michael',1.0),('Enzo',1.0),('Nouhou',1.0),('Olivier',1.0),('Pierre',1.0),('Frank',1.0),
    ('Georges',1.0),('Herve',1.0),('Jerome',1.0),('Landry',1.0),('Marius',1.0),('Norbert',1.0),
    ('Patrice',1.0),('Rene',1.0),('Serge',1.0),('Thierry',1.0),('Achille',1.0),('Bertrand',1.0),
    ('Cedric',1.0),('Didier',1.0),('Emile',1.0),('Florent',1.0),('Guy',1.0),('Hilaire',1.0),
    ('Innocent',1.0),('Josue',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_cm);

INSERT INTO lastnames_cm (lastname, freq)
SELECT * FROM (VALUES
    ('Etoo',1.0),('Song',1.0),('Aboubakar',1.0),('Onana',1.0),('Fai',1.0),('Zambo',1.0),
    ('Ngamaleu',1.0),('Anguissa',1.0),('Choupo-Moting',1.0),('Bassogog',1.0),('Mbeumo',1.0),('Ekambi',1.0),
    ('Kunde',1.0),('Mbekeli',1.0),('Ondoua',1.0),('Wooh',1.0),('Castelletto',1.0),
    ('Ngoumou',1.0),('Nkoulou',1.0),('Bahoken',1.0),('Banana',1.0),('Bikoro',1.0),('Ekanga',1.0),
    ('Ekwalla',1.0),('Foe',1.0),('Kome',1.0),('Manga',1.0),('Meyong',1.0),('Ndip',1.0),
    ('Nkoudou',1.0),('Nnomo',1.0),('Nsame',1.0),('Ottou',1.0),('Owona',1.0),('Poka',1.0),
    ('Talom',1.0),('Yombi',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_cm);

INSERT INTO names_sn ("name", freq)
SELECT * FROM (VALUES
    ('Sadio',1.0),('Kalidou',1.0),('Idrissa',1.0),('Ismaila',1.0),('Cheikhou',1.0),('Edouard',1.0),
    ('Bouna',1.0),('Salif',1.0),('Papa',1.0),('Krepin',1.0),('Nampalys',1.0),('Boulaye',1.0),
    ('Habib',1.0),('Youssouf',1.0),('Moussa',1.0),('Abdou',1.0),('Alfred',1.0),('Amath',1.0),
    ('Badou',1.0),('Cheikh',1.0),('Demba',1.0),('Famara',1.0),('Gana',1.0),('Habibou',1.0),
    ('Ibou',1.0),('Khadim',1.0),('Lamine',1.0),('Mame',1.0),('Ndiaga',1.0),('Oumar',1.0),
    ('Pathe',1.0),('Racine',1.0),('Saliou',1.0),('Tapha',1.0),('Youssoupha',1.0),('Aliou',1.0),('Bacary',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_sn);

INSERT INTO lastnames_sn (lastname, freq)
SELECT * FROM (VALUES
    ('Mane',1.0),('Koulibaly',1.0),('Gueye',1.0),('Sarr',1.0),('Kouyate',1.0),('Mendy',1.0),
    ('Sabaly',1.0),('Diallo',1.0),('Sy',1.0),('Cisse',1.0),('Ciss',1.0),('Faye',1.0),
    ('Diagne',1.0),('Dieng',1.0),('Fall',1.0),('Ndiaye',1.0),('Thiam',1.0),('Ba',1.0),
    ('Diatta',1.0),('Diedhiou',1.0),('Diop',1.0),('Fofana',1.0),('Gomis',1.0),('Jakobs',1.0),
    ('Kane',1.0),('Konate',1.0),('Ndao',1.0),('Niang',1.0),('Sane',1.0),('Seck',1.0),
    ('Sonko',1.0),('Sow',1.0),('Thioune',1.0),('Wague',1.0),('Balde',1.0),('Toure',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_sn);

INSERT INTO names_za ("name", freq)
SELECT * FROM (VALUES
    ('Percy',1.0),('Themba',1.0),('Bongani',1.0),('Thulani',1.0),('Siphiwe',1.0),('Itumeleng',1.0),
    ('Andile',1.0),('Siyabonga',1.0),('Lebogang',1.0),('Thembinkosi',1.0),('Ronwen',1.0),('Keagan',1.0),
    ('Sipho',1.0),('Vincent',1.0),('Innocent',1.0),('Hlompho',1.0),('Rulani',1.0),('Jabu',1.0),
    ('Katlego',1.0),('Lyle',1.0),('Mothobi',1.0),('Njabulo',1.0),('Oswin',1.0),('Peter',1.0),
    ('Quinton',1.0),('Rowen',1.0),('Steven',1.0),('Teboho',1.0),('Wandile',1.0),('Aubrey',1.0),
    ('Bafana',1.0),('Charlton',1.0),('Dumisani',1.0),('Elias',1.0),('Fortune',1.0),('Given',1.0),
    ('Hendrick',1.0),('Isaac',1.0),('Jomo',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_za);

INSERT INTO lastnames_za (lastname, freq)
SELECT * FROM (VALUES
    ('Tau',1.0),('Zwane',1.0),('Zungu',1.0),('Dolly',1.0),('Foster',1.0),('Mokoena',1.0),
    ('Sirino',1.0),('Khune',1.0),('Mokwena',1.0),('Ndlovu',1.0),('Modise',1.0),('Ngcongca',1.0),
    ('Kekana',1.0),('Mahlangu',1.0),('Masilela',1.0),('Ngobese',1.0),('Rantie',1.0),('Serero',1.0),
    ('Shalulile',1.0),('Sithole',1.0),('Xulu',1.0),('Zulu',1.0),('Baloyi',1.0),('Chabalala',1.0),
    ('Dladla',1.0),('Gumede',1.0),('Hlatshwayo',1.0),('Jali',1.0),('Khumalo',1.0),
    ('Maluleke',1.0),('Nhlapo',1.0),('Nyathi',1.0),('Radebe',1.0),('Sikhosana',1.0),('Sithebe',1.0),
    ('Tshabalala',1.0),('Vilakazi',1.0),('Xoki',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_za);

INSERT INTO names_ma ("name", freq)
SELECT * FROM (VALUES
    ('Achraf',1.0),('Sofyan',1.0),('Hakim',1.0),('Youssef',1.0),('Noussair',1.0),('Yassine',1.0),
    ('Ayoub',1.0),('Romain',1.0),('Nayef',1.0),('Zakaria',1.0),('Selim',1.0),('Bilal',1.0),
    ('Ilias',1.0),('Munir',1.0),('Adam',1.0),('Amine',1.0),('Badr',1.0),('Chamseddine',1.0),
    ('Driss',1.0),('Ezzine',1.0),('Farid',1.0),('Ghali',1.0),('Hamza',1.0),('Ismail',1.0),
    ('Jamal',1.0),('Karim',1.0),('Lahcen',1.0),('Mehdi',1.0),('Nabil',1.0),('Omar',1.0),
    ('Qassim',1.0),('Rachid',1.0),('Said',1.0),('Tarik',1.0),('Walid',1.0),('Yassir',1.0),
    ('Zaid',1.0),('Aymane',1.0),('Brahim',1.0),('Chakib',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ma);

INSERT INTO lastnames_ma (lastname, freq)
SELECT * FROM (VALUES
    ('Hakimi',1.0),('Ziyech',1.0),('En-Nesyri',1.0),('Boufal',1.0),('Mazraoui',1.0),('Saiss',1.0),
    ('Aguerd',1.0),('Ounahi',1.0),('Amrabat',1.0),('Bounou',1.0),('Dari',1.0),('Cheddira',1.0),
    ('Zabiri',1.0),('El Kaabi',1.0),('Benoun',1.0),('Fajr',1.0),('Hamdallah',1.0),('Jabrane',1.0),
    ('Loutfi',1.0),('Masrour',1.0),('Nassiri',1.0),('Ouahbi',1.0),('Regragui',1.0),('Sabiri',1.0),
    ('Tagnaouti',1.0),('Wahbi',1.0),('Aboukhlal',1.0),('Belhanda',1.0),('Chamakh',1.0),('Dirar',1.0),
    ('El Ahmadi',1.0),('Fellaini',1.0),('Ghanam',1.0),('Harit',1.0),('Idrissi',1.0),('Jbari',1.0),
    ('Karam',1.0),('Lamrissi',1.0),('Meski',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ma);

INSERT INTO names_dz ("name", freq)
SELECT * FROM (VALUES
    ('Riyad',1.0),('Ismael',1.0),('Islam',1.0),('Baghdad',1.0),('Yacine',1.0),('Sofiane',1.0),
    ('Youcef',1.0),('Ramy',1.0),('Rachid',1.0),('Aissa',1.0),('Djamel',1.0),('Ahmed',1.0),
    ('Nabil',1.0),('Sid',1.0),('Adem',1.0),('Bilal',1.0),('Chems',1.0),('Farid',1.0),
    ('Hocine',1.0),('Ilyes',1.0),('Karim',1.0),('Larbi',1.0),('Mehdi',1.0),('Nassim',1.0),
    ('Omar',1.0),('Rabah',1.0),('Salim',1.0),('Tayeb',1.0),('Walid',1.0),('Yassine',1.0),
    ('Zakaria',1.0),('Abdelkader',1.0),('Brahim',1.0),('Cherif',1.0),('Djaffar',1.0),('Fouad',1.0),
    ('Hakim',1.0),('Idir',1.0),('Jugurtha',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_dz);

INSERT INTO lastnames_dz (lastname, freq)
SELECT * FROM (VALUES
    ('Mahrez',1.0),('Slimani',1.0),('Bennacer',1.0),('Belaili',1.0),('Delandri',1.0),('Feghouli',1.0),
    ('Bounedjah',1.0),('Boudaoui',1.0),('Benlamri',1.0),('Atal',1.0),('Mandi',1.0),('Brahimi',1.0),
    ('Belkebla',1.0),('Guedioura',1.0),('Bentaleb',1.0),('Zerrouki',1.0),('Bensebaini',1.0),('Kacher',1.0),
    ('Aouar',1.0),('Boudebouz',1.0),('Zorgane',1.0),('Yahia',1.0),('Bezzaz',1.0),('Chetti',1.0),
    ('Djabou',1.0),('Ferhat',1.0),('Ghezzal',1.0),('Hanni',1.0),('Ighil',1.0),('Kadir',1.0),
    ('Meziane',1.0),('Naidji',1.0),('Ounas',1.0),('Rekik',1.0),('Saidi',1.0),('Taider',1.0),
    ('Younes',1.0),('Zaidi',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_dz);

INSERT INTO names_ke ("name", freq)
SELECT * FROM (VALUES
    ('Michael',1.0),('Victor',1.0),('Johanna',1.0),('Musa',1.0),('Erick',1.0),('Francis',1.0),('Duke',1.0),
    ('Timothy',1.0),('Boniface',1.0),('Eric',1.0),('Joseph',1.0),('Lawrence',1.0),('Ayub',1.0),
    ('Clifton',1.0),('Enock',1.0),('Kepha',1.0),('Masoud',1.0),('Vincent',1.0),
    ('Abud',1.0),('Bernard',1.0),('Collins',1.0),('David',1.0),('Elijah',1.0),('Felix',1.0),
    ('Gilbert',1.0),('Harun',1.0),('Isaiah',1.0),('Jackson',1.0),('Kelvin',1.0),('Levis',1.0),
    ('Moses',1.0),('Nicholas',1.0),('Oscar',1.0),('Patrick',1.0),('Robert',1.0),('Samuel',1.0),
    ('Titus',1.0),('Wesley',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ke);

INSERT INTO lastnames_ke (lastname, freq)
SELECT * FROM (VALUES
    ('Wanyama',1.0),('Olunga',1.0),('Ayimba',1.0),('Origi',1.0),('Mariga',1.0),('Otieno',1.0),
    ('Odhiambo',1.0),('Onyango',1.0),('Wafula',1.0),('Kioko',1.0),('Mutiso',1.0),('Wekesa',1.0),
    ('Simiyu',1.0),('Odongo',1.0),('Ochieng',1.0),('Njoroge',1.0),('Kamau',1.0),('Mwangi',1.0),
    ('Kariuki',1.0),('Njenga',1.0),('Kimani',1.0),('Karanja',1.0),('Waweru',1.0),('Gitau',1.0),
    ('Macharia',1.0),('Wachira',1.0),('Njuguna',1.0),('Ndungu',1.0),('Kigen',1.0),('Rotich',1.0),
    ('Kiprop',1.0),('Cheruiyot',1.0),('Korir',1.0),('Ruto',1.0),('Sang',1.0),('Barasa',1.0),
    ('Wamalwa',1.0),('Nyongesa',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ke);
