-- Adds 12 more countries (issue: "add more countries for leagues"), 3 per region (see
-- common.RegionForCountry), rounding every region out further the same way
-- init55_expand_countries.sql did. Each needs its own names_<code>/lastnames_<code> name-pool
-- pair (see common.GetRandomName/nameTableSuffixByCountry) - team/stadium naming data lives in Go
-- (see common/teamnames.go, common/stadiumnames.go), not the database. Countries stay inactive
-- (not in the `countries` table) until an admin activates one via POST /v1/admin/countries.
--
-- Every INSERT below is guarded by "WHERE NOT EXISTS (SELECT 1 FROM <table>)" so this migration
-- stays safe to re-apply on every startup (see common.EnsureSchema/applyPendingMigrations): the
-- first run seeds an empty table, every run after that sees existing rows and no-ops.

CREATE TABLE IF NOT EXISTS names_be ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_be (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_tr ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_tr (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_at ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_at (lastname varchar(100) NULL, freq float8 NULL);

CREATE TABLE IF NOT EXISTS names_ve ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ve (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_py ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_py (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_bo ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_bo (lastname varchar(100) NULL, freq float8 NULL);

CREATE TABLE IF NOT EXISTS names_qa ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_qa (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_ae ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ae (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_uz ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_uz (lastname varchar(100) NULL, freq float8 NULL);

CREATE TABLE IF NOT EXISTS names_tn ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_tn (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_tz ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_tz (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_ml ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ml (lastname varchar(100) NULL, freq float8 NULL);

INSERT INTO names_be ("name", freq)
SELECT * FROM (VALUES
    ('Lucas',1.0),('Liam',1.0),('Noah',1.0),('Finn',1.0),('Sem',1.0),('Jonas',1.0),('Thomas',1.0),
    ('Nathan',1.0),('Milan',1.0),('Lars',1.0),('Dries',1.0),('Kobe',1.0),('Tom',1.0),('Wout',1.0),
    ('Stijn',1.0),('Bram',1.0),('Senne',1.0),('Tuur',1.0),('Mathis',1.0),('Victor',1.0),('Hugo',1.0),
    ('Louis',1.0),('Arthur',1.0),('Axel',1.0),('Jens',1.0),('Niels',1.0),('Kevin',1.0),('Maxime',1.0),
    ('Simon',1.0),('Florian',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_be);

INSERT INTO lastnames_be (lastname, freq)
SELECT * FROM (VALUES
    ('Peeters',1.0),('Janssens',1.0),('Maes',1.0),('Jacobs',1.0),('Mertens',1.0),('Willems',1.0),
    ('Claes',1.0),('Goossens',1.0),('Wouters',1.0),('De Smet',1.0),('Dubois',1.0),('Lambert',1.0),
    ('Dupont',1.0),('Lemmens',1.0),('Hermans',1.0),('Pauwels',1.0),('Verhoeven',1.0),('Vermeulen',1.0),
    ('Janssen',1.0),('Bogaert',1.0),('Van Damme',1.0),('Vandenberghe',1.0),('De Clercq',1.0),
    ('Michiels',1.0),('Desmet',1.0),('Lefevre',1.0),('Simon',1.0),('Henry',1.0),('Collard',1.0),('Smet',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_be);

INSERT INTO names_tr ("name", freq)
SELECT * FROM (VALUES
    ('Mehmet',1.0),('Mustafa',1.0),('Ahmet',1.0),('Ali',1.0),('Huseyin',1.0),('Hasan',1.0),
    ('Ibrahim',1.0),('Yusuf',1.0),('Emre',1.0),('Burak',1.0),('Serkan',1.0),('Ozan',1.0),('Caner',1.0),
    ('Kerem',1.0),('Baris',1.0),('Onur',1.0),('Murat',1.0),('Deniz',1.0),('Umut',1.0),('Fatih',1.0),
    ('Cem',1.0),('Selim',1.0),('Tolga',1.0),('Kaan',1.0),('Volkan',1.0),('Gokhan',1.0),('Eren',1.0),
    ('Arda',1.0),('Can',1.0),('Berk',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_tr);

INSERT INTO lastnames_tr (lastname, freq)
SELECT * FROM (VALUES
    ('Yilmaz',1.0),('Kaya',1.0),('Demir',1.0),('Sahin',1.0),('Celik',1.0),('Yildiz',1.0),
    ('Yildirim',1.0),('Ozturk',1.0),('Aydin',1.0),('Ozdemir',1.0),('Arslan',1.0),('Dogan',1.0),
    ('Kilic',1.0),('Aslan',1.0),('Cetin',1.0),('Kara',1.0),('Koc',1.0),('Kurt',1.0),('Ozkan',1.0),
    ('Simsek',1.0),('Polat',1.0),('Korkmaz',1.0),('Cakir',1.0),('Candan',1.0),('Erdogan',1.0),
    ('Akin',1.0),('Tekin',1.0),('Bulut',1.0),('Gunes',1.0),('Acar',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_tr);

INSERT INTO names_at ("name", freq)
SELECT * FROM (VALUES
    ('Lukas',1.0),('Felix',1.0),('David',1.0),('Jakob',1.0),('Simon',1.0),('Florian',1.0),
    ('Michael',1.0),('Matthias',1.0),('Daniel',1.0),('Fabian',1.0),('Christoph',1.0),('Stefan',1.0),
    ('Thomas',1.0),('Markus',1.0),('Philipp',1.0),('Andreas',1.0),('Martin',1.0),('Alexander',1.0),
    ('Benjamin',1.0),('Julian',1.0),('Paul',1.0),('Tobias',1.0),('Maximilian',1.0),('Sebastian',1.0),
    ('Dominik',1.0),('Patrick',1.0),('Clemens',1.0),('Georg',1.0),('Niklas',1.0),('Moritz',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_at);

INSERT INTO lastnames_at (lastname, freq)
SELECT * FROM (VALUES
    ('Gruber',1.0),('Huber',1.0),('Bauer',1.0),('Wagner',1.0),('Mueller',1.0),('Pichler',1.0),
    ('Steiner',1.0),('Moser',1.0),('Mayer',1.0),('Hofer',1.0),('Leitner',1.0),('Berger',1.0),
    ('Fuchs',1.0),('Eder',1.0),('Fischer',1.0),('Schmid',1.0),('Winkler',1.0),('Weber',1.0),
    ('Schwarz',1.0),('Maier',1.0),('Wimmer',1.0),('Egger',1.0),('Brunner',1.0),('Lang',1.0),
    ('Baumgartner',1.0),('Auer',1.0),('Reiter',1.0),('Mayr',1.0),('Schneider',1.0),('Wolf',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_at);

INSERT INTO names_ve ("name", freq)
SELECT * FROM (VALUES
    ('Carlos',1.0),('Jose',1.0),('Luis',1.0),('Miguel',1.0),('Juan',1.0),('Alejandro',1.0),
    ('Francisco',1.0),('Rafael',1.0),('Antonio',1.0),('Eduardo',1.0),('Gustavo',1.0),('Ricardo',1.0),
    ('Oswaldo',1.0),('Victor',1.0),('Pedro',1.0),('Manuel',1.0),('Ramon',1.0),('Jesus',1.0),
    ('Angel',1.0),('Cesar',1.0),('Freddy',1.0),('Alberto',1.0),('Wilmer',1.0),('Yonathan',1.0),
    ('Elvis',1.0),('Yoel',1.0),('Gerardo',1.0),('Ender',1.0),('Darwin',1.0),('Salomon',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ve);

INSERT INTO lastnames_ve (lastname, freq)
SELECT * FROM (VALUES
    ('Gonzalez',1.0),('Rodriguez',1.0),('Perez',1.0),('Martinez',1.0),('Hernandez',1.0),('Lopez',1.0),
    ('Garcia',1.0),('Diaz',1.0),('Rojas',1.0),('Castillo',1.0),('Sanchez',1.0),('Ramirez',1.0),
    ('Torres',1.0),('Flores',1.0),('Marin',1.0),('Mendoza',1.0),('Vargas',1.0),('Guerra',1.0),
    ('Pacheco',1.0),('Escalona',1.0),('Aguilar',1.0),('Blanco',1.0),('Rosales',1.0),('Medina',1.0),
    ('Carrero',1.0),('Bello',1.0),('Pino',1.0),('Prado',1.0),('Zambrano',1.0),('Colina',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ve);

INSERT INTO names_py ("name", freq)
SELECT * FROM (VALUES
    ('Diego',1.0),('Gustavo',1.0),('Ramon',1.0),('Victor',1.0),('Nestor',1.0),('Cristian',1.0),
    ('Richard',1.0),('Pedro',1.0),('Ruben',1.0),('Pablo',1.0),('Miguel',1.0),('Dario',1.0),('Hugo',1.0),
    ('Sergio',1.0),('Oscar',1.0),('Juan',1.0),('Antonio',1.0),('Carlos',1.0),('Fernando',1.0),
    ('Alcides',1.0),('Julio',1.0),('Nelson',1.0),('Walter',1.0),('Osvaldo',1.0),('Angel',1.0),
    ('Ever',1.0),('Derlis',1.0),('Celso',1.0),('Gabriel',1.0),('Marcelo',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_py);

INSERT INTO lastnames_py (lastname, freq)
SELECT * FROM (VALUES
    ('Gonzalez',1.0),('Benitez',1.0),('Rolon',1.0),('Ortiz',1.0),('Cardozo',1.0),('Villalba',1.0),
    ('Sanabria',1.0),('Acosta',1.0),('Caceres',1.0),('Ferreira',1.0),('Martinez',1.0),('Fleitas',1.0),
    ('Ayala',1.0),('Barreto',1.0),('Melgarejo',1.0),('Duarte',1.0),('Franco',1.0),('Riveros',1.0),
    ('Zarate',1.0),('Gimenez',1.0),('Encina',1.0),('Vera',1.0),('Paredes',1.0),('Samudio',1.0),
    ('Bogado',1.0),('Arrua',1.0),('Insfran',1.0),('Godoy',1.0),('Candia',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_py);

INSERT INTO names_bo ("name", freq)
SELECT * FROM (VALUES
    ('Marcelo',1.0),('Raul',1.0),('Ronald',1.0),('Diego',1.0),('Pablo',1.0),('Juan',1.0),('Luis',1.0),
    ('Erwin',1.0),('Ramiro',1.0),('Boris',1.0),('Edward',1.0),('Henry',1.0),('Franz',1.0),('Alvaro',1.0),
    ('Carmelo',1.0),('Walter',1.0),('Fernando',1.0),('Gualberto',1.0),('Victor',1.0),('Gustavo',1.0),
    ('Jose',1.0),('Carlos',1.0),('Mauricio',1.0),('Jesus',1.0),('Richard',1.0),('Leonel',1.0),
    ('Robson',1.0),('Moises',1.0),('Adrian',1.0),('Jhasmani',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_bo);

INSERT INTO lastnames_bo (lastname, freq)
SELECT * FROM (VALUES
    ('Quiroga',1.0),('Mamani',1.0),('Choque',1.0),('Flores',1.0),('Rivero',1.0),('Vargas',1.0),
    ('Pena',1.0),('Rojas',1.0),('Terrazas',1.0),('Chavez',1.0),('Saavedra',1.0),('Villarroel',1.0),
    ('Justiniano',1.0),('Arce',1.0),('Ribera',1.0),('Suarez',1.0),('Melgar',1.0),('Duran',1.0),
    ('Rocabado',1.0),('Cespedes',1.0),('Condori',1.0),('Rodriguez',1.0),('Gutierrez',1.0),
    ('Moreno',1.0),('Roman',1.0),('Martinez',1.0),('Ortega',1.0),('Zurita',1.0),('Alanoca',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_bo);

INSERT INTO names_qa ("name", freq)
SELECT * FROM (VALUES
    ('Mohammed',1.0),('Ahmed',1.0),('Abdullah',1.0),('Khalid',1.0),('Hamad',1.0),('Ali',1.0),
    ('Rashid',1.0),('Saoud',1.0),('Jassim',1.0),('Nasser',1.0),('Faisal',1.0),('Tariq',1.0),
    ('Yousuf',1.0),('Fahad',1.0),('Majed',1.0),('Saad',1.0),('Salem',1.0),('Omar',1.0),('Hassan',1.0),
    ('Turki',1.0),('Essa',1.0),('Abdulrahman',1.0),('Mubarak',1.0),('Ghanim',1.0),('Mansour',1.0),
    ('Sultan',1.0),('Marzouq',1.0),('Hilal',1.0),('Khalifa',1.0),('Thani',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_qa);

INSERT INTO lastnames_qa (lastname, freq)
SELECT * FROM (VALUES
    ('Al-Thani',1.0),('Al-Kaabi',1.0),('Al-Marri',1.0),('Al-Sulaiti',1.0),('Al-Ansari',1.0),
    ('Al-Hajri',1.0),('Al-Khalifa',1.0),('Al-Obaidli',1.0),('Al-Mannai',1.0),('Al-Mohannadi',1.0),
    ('Al-Sayed',1.0),('Al-Yafei',1.0),('Al-Suwaidi',1.0),('Al-Naimi',1.0),('Al-Shammari',1.0),
    ('Al-Emadi',1.0),('Al-Baker',1.0),('Al-Attiyah',1.0),('Al-Khater',1.0),('Al-Darwish',1.0),
    ('Al-Jaber',1.0),('Al-Rumaihi',1.0),('Al-Nuaimi',1.0),('Al-Mahmoud',1.0),('Al-Qahtani',1.0),
    ('Al-Dosari',1.0),('Al-Mulla',1.0),('Al-Hitmi',1.0),('Al-Sada',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_qa);

INSERT INTO names_ae ("name", freq)
SELECT * FROM (VALUES
    ('Mohammed',1.0),('Ahmed',1.0),('Rashid',1.0),('Hamdan',1.0),('Saeed',1.0),('Khalifa',1.0),
    ('Sultan',1.0),('Omar',1.0),('Hamad',1.0),('Abdullah',1.0),('Faisal',1.0),('Majid',1.0),
    ('Khalid',1.0),('Tariq',1.0),('Saif',1.0),('Nasser',1.0),('Mansour',1.0),('Hilal',1.0),
    ('Yousuf',1.0),('Ali',1.0),('Humaid',1.0),('Zayed',1.0),('Butti',1.0),('Dhiyab',1.0),
    ('Marwan',1.0),('Juma',1.0),('Obaid',1.0),('Rakan',1.0),('Hazim',1.0),('Salem',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ae);

INSERT INTO lastnames_ae (lastname, freq)
SELECT * FROM (VALUES
    ('Al-Maktoum',1.0),('Al-Nahyan',1.0),('Al-Qasimi',1.0),('Al-Nuaimi',1.0),('Al-Shamsi',1.0),
    ('Al-Zaabi',1.0),('Al-Mansouri',1.0),('Al-Suwaidi',1.0),('Al-Falasi',1.0),('Al-Kaabi',1.0),
    ('Al-Hashimi',1.0),('Al-Marri',1.0),('Al-Ketbi',1.0),('Al-Dhaheri',1.0),('Al-Mazrouei',1.0),
    ('Al-Rashidi',1.0),('Al-Ameri',1.0),('Al-Hameli',1.0),('Al-Blooshi',1.0),('Al-Habsi',1.0),
    ('Al-Qubaisi',1.0),('Al-Hammadi',1.0),('Al-Mehrezi',1.0),('Al-Awadhi',1.0),('Al-Jaberi',1.0),
    ('Al-Shehhi',1.0),('Al-Mulla',1.0),('Al-Romaithi',1.0),('Al-Zarouni',1.0),('Al-Yamahi',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ae);

INSERT INTO names_uz ("name", freq)
SELECT * FROM (VALUES
    ('Islom',1.0),('Jasur',1.0),('Sardor',1.0),('Bekzod',1.0),('Dilshod',1.0),('Farrukh',1.0),
    ('Shohrux',1.0),('Rustam',1.0),('Otabek',1.0),('Javlon',1.0),('Sanjar',1.0),('Bobur',1.0),
    ('Aziz',1.0),('Jamshid',1.0),('Alisher',1.0),('Timur',1.0),('Davron',1.0),('Shavkat',1.0),
    ('Ulugbek',1.0),('Behzod',1.0),('Muzaffar',1.0),('Anvar',1.0),('Sherzod',1.0),('Nodir',1.0),
    ('Ravshan',1.0),('Komil',1.0),('Rasul',1.0),('Firdavs',1.0),('Umid',1.0),('Zafar',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_uz);

INSERT INTO lastnames_uz (lastname, freq)
SELECT * FROM (VALUES
    ('Karimov',1.0),('Yusupov',1.0),('Rashidov',1.0),('Islomov',1.0),('Abdullaev',1.0),
    ('Nazarov',1.0),('Yuldashev',1.0),('Rustamov',1.0),('Saidov',1.0),('Ergashev',1.0),
    ('Tursunov',1.0),('Mirzaev',1.0),('Khodjaev',1.0),('Sultanov',1.0),('Nuraliev',1.0),
    ('Bekmuradov',1.0),('Jumaev',1.0),('Khalilov',1.0),('Shukurov',1.0),('Pulatov',1.0),
    ('Egamberdiev',1.0),('Usmanov',1.0),('Fayzullaev',1.0),('Toirov',1.0),('Qosimov',1.0),
    ('Berdiev',1.0),('Xolmatov',1.0),('Ismoilov',1.0),('Ochilov',1.0),('Tashkentov',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_uz);

INSERT INTO names_tn ("name", freq)
SELECT * FROM (VALUES
    ('Mohamed',1.0),('Ahmed',1.0),('Youssef',1.0),('Karim',1.0),('Sami',1.0),('Walid',1.0),
    ('Nabil',1.0),('Hichem',1.0),('Anis',1.0),('Rami',1.0),('Bilel',1.0),('Khalil',1.0),('Mehdi',1.0),
    ('Amine',1.0),('Oussama',1.0),('Marouane',1.0),('Firas',1.0),('Zied',1.0),('Tarek',1.0),
    ('Riadh',1.0),('Chiheb',1.0),('Moez',1.0),('Fares',1.0),('Montassar',1.0),('Haythem',1.0),
    ('Aymen',1.0),('Souhail',1.0),('Hamza',1.0),('Skander',1.0),('Bassem',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_tn);

INSERT INTO lastnames_tn (lastname, freq)
SELECT * FROM (VALUES
    ('Ben Ali',1.0),('Trabelsi',1.0),('Jebali',1.0),('Mabrouki',1.0),('Gharbi',1.0),('Hamdi',1.0),
    ('Mansouri',1.0),('Saidi',1.0),('Bouazizi',1.0),('Khlifi',1.0),('Mejri',1.0),('Belhaj',1.0),
    ('Kallel',1.0),('Chaabane',1.0),('Souissi',1.0),('Bouzid',1.0),('Ferjani',1.0),('Zaouali',1.0),
    ('Dridi',1.0),('Naceur',1.0),('Haddad',1.0),('Khalfallah',1.0),('Riahi',1.0),('Mnif',1.0),
    ('Mejdoub',1.0),('Bousnina',1.0),('Sahli',1.0),('Toumi',1.0),('Ayari',1.0),('Jlassi',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_tn);

INSERT INTO names_tz ("name", freq)
SELECT * FROM (VALUES
    ('Juma',1.0),('Hassan',1.0),('Said',1.0),('Ally',1.0),('Mussa',1.0),('Ibrahim',1.0),
    ('Rashid',1.0),('Omari',1.0),('Salum',1.0),('Hamisi',1.0),('Abdallah',1.0),('Mwinyi',1.0),
    ('Yahya',1.0),('Shabani',1.0),('Issa',1.0),('Athumani',1.0),('Bakari',1.0),('Idrisa',1.0),
    ('Selemani',1.0),('Kassim',1.0),('Daudi',1.0),('Baraka',1.0),('Emmanuel',1.0),('Fredy',1.0),
    ('Godfrey',1.0),('Elias',1.0),('Mathias',1.0),('Erasto',1.0),('Deogratias',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_tz);

INSERT INTO lastnames_tz (lastname, freq)
SELECT * FROM (VALUES
    ('Mwakalinga',1.0),('Kimaro',1.0),('Chacha',1.0),('Mushi',1.0),('Mrema',1.0),('Nyoni',1.0),
    ('Mnyamani',1.0),('Lyimo',1.0),('Shirima',1.0),('Mallya',1.0),('Massawe',1.0),('Kweka',1.0),
    ('Komba',1.0),('Mbwambo',1.0),('Sanga',1.0),('Mandara',1.0),('Haule',1.0),('Kapinga',1.0),
    ('Mwamba',1.0),('Kessy',1.0),('Lugendo',1.0),('Mbise',1.0),('Rutta',1.0),('Kivuyo',1.0),
    ('Mwampamba',1.0),('Mahundi',1.0),('Kinabo',1.0),('Mushaija',1.0),('Temba',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_tz);

INSERT INTO names_ml ("name", freq)
SELECT * FROM (VALUES
    ('Oumar',1.0),('Moussa',1.0),('Ibrahim',1.0),('Sekou',1.0),('Mamadou',1.0),('Boubacar',1.0),
    ('Modibo',1.0),('Drissa',1.0),('Amadou',1.0),('Yacouba',1.0),('Lassana',1.0),('Bakary',1.0),
    ('Adama',1.0),('Souleymane',1.0),('Issa',1.0),('Cheick',1.0),('Alou',1.0),('Mahamadou',1.0),
    ('Fousseyni',1.0),('Karim',1.0),('Siaka',1.0),('Abdoulaye',1.0),('Youssouf',1.0),('Daouda',1.0),
    ('Zoumana',1.0),('Toumani',1.0),('Hamidou',1.0),('Seydou',1.0),('Kassim',1.0),('Baba',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ml);

INSERT INTO lastnames_ml (lastname, freq)
SELECT * FROM (VALUES
    ('Traore',1.0),('Diarra',1.0),('Keita',1.0),('Coulibaly',1.0),('Diallo',1.0),('Toure',1.0),
    ('Cisse',1.0),('Konate',1.0),('Sangare',1.0),('Doumbia',1.0),('Kone',1.0),('Camara',1.0),
    ('Sidibe',1.0),('Sanogo',1.0),('Fofana',1.0),('Dembele',1.0),('Haidara',1.0),('Tounkara',1.0),
    ('Bagayogo',1.0),('Samake',1.0),('Maiga',1.0),('Diakite',1.0),('Kante',1.0),('Sow',1.0),
    ('Tangara',1.0),('Kamissoko',1.0),('Djire',1.0),('Diabate',1.0),('Sissoko',1.0),('Niangadou',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ml);
