-- Adds 11 more countries - the three Celtic footballing nations plus Russia and the countries of
-- the former Yugoslavia (see common.RegionForCountry, all grouped under "europe"), rounding
-- Europe out further the same way init55_expand_countries.sql/init68_more_countries.sql did.
-- Each needs its own names_<code>/lastnames_<code> name-pool pair (see common.GetRandomName/
-- nameTableSuffixByCountry) - team/stadium naming data lives in Go (see common/teamnames.go,
-- common/stadiumnames.go), not the database. Countries stay inactive (not in the `countries`
-- table) until an admin activates one via POST /v1/admin/countries.
--
-- Scotland/Wales use the invented 3-letter codes "sco"/"wal" (no ISO 3166-1 code exists for
-- either - they're separate footballing nations, not separate states); every other code here is
-- a real ISO 3166-1 alpha-2 code, except Kosovo's "xk" (the common unofficial/user-assigned code,
-- since Kosovo has no ISO code either).
--
-- Every INSERT below is guarded by "WHERE NOT EXISTS (SELECT 1 FROM <table>)" so this migration
-- stays safe to re-apply on every startup (see common.EnsureSchema/applyPendingMigrations): the
-- first run seeds an empty table, every run after that sees existing rows and no-ops.

CREATE TABLE IF NOT EXISTS names_sco ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_sco (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_wal ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_wal (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_ie ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ie (lastname varchar(100) NULL, freq float8 NULL);

CREATE TABLE IF NOT EXISTS names_ru ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ru (lastname varchar(100) NULL, freq float8 NULL);

CREATE TABLE IF NOT EXISTS names_rs ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_rs (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_hr ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_hr (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_ba ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_ba (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_si ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_si (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_mk ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_mk (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_me ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_me (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_xk ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_xk (lastname varchar(100) NULL, freq float8 NULL);

INSERT INTO names_sco ("name", freq)
SELECT * FROM (VALUES
    ('Jack',1.0),('Lewis',1.0),('Ryan',1.0),('Callum',1.0),('Euan',1.0),('Fraser',1.0),
    ('Angus',1.0),('Rory',1.0),('Cameron',1.0),('Ewan',1.0),('Liam',1.0),('Connor',1.0),
    ('Gordon',1.0),('Hamish',1.0),('Alistair',1.0),('Craig',1.0),('Scott',1.0),('Kyle',1.0),
    ('Robbie',1.0),('Stuart',1.0),('Graeme',1.0),('Keith',1.0),('Ian',1.0),('Murray',1.0),
    ('Duncan',1.0),('Ross',1.0),('Calum',1.0),('Jamie',1.0),('Finlay',1.0),('Innes',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_sco);

INSERT INTO lastnames_sco (lastname, freq)
SELECT * FROM (VALUES
    ('MacLeod',1.0),('MacDonald',1.0),('Campbell',1.0),('Stewart',1.0),('Robertson',1.0),
    ('Morrison',1.0),('Fraser',1.0),('Grant',1.0),('Cameron',1.0),('Wallace',1.0),
    ('Ferguson',1.0),('Henderson',1.0),('Murray',1.0),('Watson',1.0),('Kerr',1.0),
    ('Reid',1.0),('Scott',1.0),('Mackay',1.0),('Ross',1.0),('Sinclair',1.0),
    ('Anderson',1.0),('Buchanan',1.0),('Gordon',1.0),('MacKenzie',1.0),('Crawford',1.0),
    ('Hamilton',1.0),('Munro',1.0),('Douglas',1.0),('Cunningham',1.0),('Baxter',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_sco);

INSERT INTO names_wal ("name", freq)
SELECT * FROM (VALUES
    ('Dylan',1.0),('Rhys',1.0),('Gareth',1.0),('Owen',1.0),('Rhodri',1.0),('Gethin',1.0),
    ('Aled',1.0),('Dafydd',1.0),('Ioan',1.0),('Carwyn',1.0),('Ifan',1.0),('Osian',1.0),
    ('Tomos',1.0),('Ellis',1.0),('Harri',1.0),('Llewelyn',1.0),('Cai',1.0),('Steffan',1.0),
    ('Garin',1.0),('Cerith',1.0),('Huw',1.0),('Emyr',1.0),('Tudur',1.0),('Bryn',1.0),
    ('Gwilym',1.0),('Rhidian',1.0),('Macsen',1.0),('Elgan',1.0),('Iestyn',1.0),('Aneurin',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_wal);

INSERT INTO lastnames_wal (lastname, freq)
SELECT * FROM (VALUES
    ('Jones',1.0),('Williams',1.0),('Davies',1.0),('Evans',1.0),('Thomas',1.0),
    ('Roberts',1.0),('Lewis',1.0),('Hughes',1.0),('Morgan',1.0),('Griffiths',1.0),
    ('Jenkins',1.0),('Rees',1.0),('Edwards',1.0),('Phillips',1.0),('Morris',1.0),
    ('Owen',1.0),('Pritchard',1.0),('Price',1.0),('Rowlands',1.0),('Vaughan',1.0),
    ('Powell',1.0),('Parry',1.0),('Pugh',1.0),('Lloyd',1.0),('Hopkins',1.0),
    ('Meredith',1.0),('Probert',1.0),('Bevan',1.0),('Pryce',1.0),('Rhys',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_wal);

INSERT INTO names_ie ("name", freq)
SELECT * FROM (VALUES
    ('Conor',1.0),('Sean',1.0),('Liam',1.0),('Patrick',1.0),('Cian',1.0),('Darragh',1.0),
    ('Eoin',1.0),('Niall',1.0),('Cillian',1.0),('Oisin',1.0),('Ronan',1.0),('Aidan',1.0),
    ('Fionn',1.0),('Diarmuid',1.0),('Shane',1.0),('Declan',1.0),('Brendan',1.0),('Donal',1.0),
    ('Padraig',1.0),('Tadhg',1.0),('Ciaran',1.0),('Kevin',1.0),('Micheal',1.0),('Colm',1.0),
    ('Barry',1.0),('Enda',1.0),('Cathal',1.0),('Rory',1.0),('Gearoid',1.0),('Brian',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ie);

INSERT INTO lastnames_ie (lastname, freq)
SELECT * FROM (VALUES
    ('Murphy',1.0),('Kelly',1.0),('Sullivan',1.0),('Walsh',1.0),('Smith',1.0),
    ('O''Brien',1.0),('Byrne',1.0),('Ryan',1.0),('O''Connor',1.0),('O''Neill',1.0),
    ('Reilly',1.0),('Doyle',1.0),('McCarthy',1.0),('Gallagher',1.0),('Kennedy',1.0),
    ('Lynch',1.0),('Murray',1.0),('Quinn',1.0),('Moore',1.0),('McLoughlin',1.0),
    ('Carroll',1.0),('Connolly',1.0),('Daly',1.0),('O''Connell',1.0),('Dunne',1.0),
    ('Brennan',1.0),('Burke',1.0),('Collins',1.0),('Clarke',1.0),('Fitzgerald',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ie);

INSERT INTO names_ru ("name", freq)
SELECT * FROM (VALUES
    ('Ivan',1.0),('Dmitri',1.0),('Alexei',1.0),('Sergei',1.0),('Mikhail',1.0),('Andrei',1.0),
    ('Nikolai',1.0),('Pavel',1.0),('Vladimir',1.0),('Yuri',1.0),('Oleg',1.0),('Viktor',1.0),
    ('Igor',1.0),('Artem',1.0),('Denis',1.0),('Roman',1.0),('Maxim',1.0),('Anton',1.0),
    ('Konstantin',1.0),('Egor',1.0),('Stanislav',1.0),('Boris',1.0),('Gennady',1.0),('Ruslan',1.0),
    ('Vadim',1.0),('Leonid',1.0),('Grigori',1.0),('Timur',1.0),('Semyon',1.0),('Arkady',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ru);

INSERT INTO lastnames_ru (lastname, freq)
SELECT * FROM (VALUES
    ('Ivanov',1.0),('Smirnov',1.0),('Kuznetsov',1.0),('Popov',1.0),('Vasiliev',1.0),
    ('Petrov',1.0),('Sokolov',1.0),('Mikhailov',1.0),('Novikov',1.0),('Fedorov',1.0),
    ('Morozov',1.0),('Volkov',1.0),('Alekseev',1.0),('Lebedev',1.0),('Semenov',1.0),
    ('Egorov',1.0),('Pavlov',1.0),('Kozlov',1.0),('Stepanov',1.0),('Nikolaev',1.0),
    ('Orlov',1.0),('Andreev',1.0),('Makarov',1.0),('Nikitin',1.0),('Zakharov',1.0),
    ('Zaytsev',1.0),('Soloviev',1.0),('Borisov',1.0),('Yakovlev',1.0),('Grigoriev',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ru);

INSERT INTO names_rs ("name", freq)
SELECT * FROM (VALUES
    ('Nikola',1.0),('Milos',1.0),('Marko',1.0),('Stefan',1.0),('Aleksandar',1.0),('Nemanja',1.0),
    ('Dusan',1.0),('Vladimir',1.0),('Bojan',1.0),('Filip',1.0),('Luka',1.0),('Ognjen',1.0),
    ('Zoran',1.0),('Dejan',1.0),('Goran',1.0),('Vuk',1.0),('Petar',1.0),('Radovan',1.0),
    ('Branislav',1.0),('Sasa',1.0),('Slobodan',1.0),('Milan',1.0),('Dragan',1.0),('Veljko',1.0),
    ('Jovan',1.0),('Andrija',1.0),('Bogdan',1.0),('Uros',1.0),('Nenad',1.0),('Rade',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_rs);

INSERT INTO lastnames_rs (lastname, freq)
SELECT * FROM (VALUES
    ('Jovanovic',1.0),('Petrovic',1.0),('Nikolic',1.0),('Markovic',1.0),('Djordjevic',1.0),
    ('Stojanovic',1.0),('Ilic',1.0),('Stankovic',1.0),('Pavlovic',1.0),('Milosevic',1.0),
    ('Zivkovic',1.0),('Vukovic',1.0),('Ristic',1.0),('Tomic',1.0),('Simic',1.0),
    ('Kovacevic',1.0),('Jankovic',1.0),('Lazic',1.0),('Radic',1.0),('Antic',1.0),
    ('Savic',1.0),('Milic',1.0),('Dimitrijevic',1.0),('Maric',1.0),('Jaksic',1.0),
    ('Vasic',1.0),('Ninkovic',1.0),('Rakic',1.0),('Obradovic',1.0),('Tadic',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_rs);

INSERT INTO names_hr ("name", freq)
SELECT * FROM (VALUES
    ('Luka',1.0),('Ivan',1.0),('Marko',1.0),('Ante',1.0),('Josip',1.0),('Filip',1.0),
    ('Mario',1.0),('Tomislav',1.0),('Petar',1.0),('Nikola',1.0),('Marin',1.0),('Domagoj',1.0),
    ('Dario',1.0),('Zoran',1.0),('Davor',1.0),('Kresimir',1.0),('Bruno',1.0),('Mislav',1.0),
    ('Sime',1.0),('Stipe',1.0),('Vedran',1.0),('Vinko',1.0),('Zvonimir',1.0),('Hrvoje',1.0),
    ('Goran',1.0),('Tihomir',1.0),('Branimir',1.0),('Mateo',1.0),('Leon',1.0),('Jure',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_hr);

INSERT INTO lastnames_hr (lastname, freq)
SELECT * FROM (VALUES
    ('Horvat',1.0),('Kovacevic',1.0),('Babic',1.0),('Markovic',1.0),('Novak',1.0),
    ('Jurkovic',1.0),('Kovac',1.0),('Vukovic',1.0),('Juric',1.0),('Peric',1.0),
    ('Matic',1.0),('Blazevic',1.0),('Kovacic',1.0),('Bosnjak',1.0),('Curic',1.0),
    ('Radic',1.0),('Simic',1.0),('Tomic',1.0),('Zoric',1.0),('Grgic',1.0),
    ('Ivankovic',1.0),('Petrovic',1.0),('Barisic',1.0),('Perkovic',1.0),('Vidovic',1.0),
    ('Lovric',1.0),('Sabic',1.0),('Pavlovic',1.0),('Rados',1.0),('Maric',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_hr);

INSERT INTO names_ba ("name", freq)
SELECT * FROM (VALUES
    ('Amar',1.0),('Haris',1.0),('Emir',1.0),('Adnan',1.0),('Kemal',1.0),('Edin',1.0),
    ('Elvir',1.0),('Nermin',1.0),('Samir',1.0),('Sead',1.0),('Zlatan',1.0),('Dino',1.0),
    ('Tarik',1.0),('Mirza',1.0),('Amel',1.0),('Senad',1.0),('Asim',1.0),('Vedad',1.0),
    ('Ismet',1.0),('Alen',1.0),('Jasmin',1.0),('Denis',1.0),('Faruk',1.0),('Zijad',1.0),
    ('Munib',1.0),('Nihad',1.0),('Rusmir',1.0),('Suad',1.0),('Ahmed',1.0),('Enes',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_ba);

INSERT INTO lastnames_ba (lastname, freq)
SELECT * FROM (VALUES
    ('Hodzic',1.0),('Begic',1.0),('Karic',1.0),('Suljic',1.0),('Mehic',1.0),
    ('Alic',1.0),('Delic',1.0),('Omerovic',1.0),('Husic',1.0),('Pasic',1.0),
    ('Ibrahimovic',1.0),('Softic',1.0),('Music',1.0),('Kadic',1.0),('Halilovic',1.0),
    ('Imamovic',1.0),('Kovacevic',1.0),('Dzafic',1.0),('Osmanovic',1.0),('Sarajlic',1.0),
    ('Beganovic',1.0),('Hamzic',1.0),('Kurtovic',1.0),('Zilic',1.0),('Memic',1.0),
    ('Durakovic',1.0),('Celic',1.0),('Avdic',1.0),('Zukic',1.0),('Sabanovic',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_ba);

INSERT INTO names_si ("name", freq)
SELECT * FROM (VALUES
    ('Jure',1.0),('Luka',1.0),('Jan',1.0),('Matej',1.0),('Tilen',1.0),('Rok',1.0),
    ('Ziga',1.0),('Nejc',1.0),('Miha',1.0),('Blaz',1.0),('Tim',1.0),('David',1.0),
    ('Marko',1.0),('Tomaz',1.0),('Primoz',1.0),('Gregor',1.0),('Andrej',1.0),('Bojan',1.0),
    ('Igor',1.0),('Uros',1.0),('Simon',1.0),('Peter',1.0),('Ales',1.0),('Bostjan',1.0),
    ('Klemen',1.0),('Domen',1.0),('Vid',1.0),('Matic',1.0),('Erik',1.0),('Dejan',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_si);

INSERT INTO lastnames_si (lastname, freq)
SELECT * FROM (VALUES
    ('Novak',1.0),('Horvat',1.0),('Kovacic',1.0),('Zupancic',1.0),('Potocnik',1.0),
    ('Krajnc',1.0),('Vidmar',1.0),('Kranjc',1.0),('Golob',1.0),('Bizjak',1.0),
    ('Mlakar',1.0),('Kolar',1.0),('Rupar',1.0),('Pirc',1.0),('Erjavec',1.0),
    ('Kastelic',1.0),('Tomsic',1.0),('Zupan',1.0),('Pavlin',1.0),('Bevc',1.0),
    ('Hribar',1.0),('Zajc',1.0),('Oblak',1.0),('Skulj',1.0),('Vovk',1.0),
    ('Dolinar',1.0),('Korosec',1.0),('Zadravec',1.0),('Plesnik',1.0),('Strnad',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_si);

INSERT INTO names_mk ("name", freq)
SELECT * FROM (VALUES
    ('Goran',1.0),('Darko',1.0),('Vlatko',1.0),('Stojan',1.0),('Zoran',1.0),('Dejan',1.0),
    ('Bojan',1.0),('Igor',1.0),('Aleksandar',1.0),('Toni',1.0),('Filip',1.0),('Marjan',1.0),
    ('Nikola',1.0),('Vanco',1.0),('Kire',1.0),('Trajce',1.0),('Petar',1.0),('Ljupco',1.0),
    ('Blagoja',1.0),('Done',1.0),('Riste',1.0),('Boban',1.0),('Sasho',1.0),('Zlatko',1.0),
    ('Dragan',1.0),('Branko',1.0),('Vasko',1.0),('Gjorgji',1.0),('Mite',1.0),('Pance',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_mk);

INSERT INTO lastnames_mk (lastname, freq)
SELECT * FROM (VALUES
    ('Stojanovski',1.0),('Petrovski',1.0),('Nikolovski',1.0),('Trajkovski',1.0),('Angelovski',1.0),
    ('Jovanovski',1.0),('Dimitrovski',1.0),('Georgievski',1.0),('Todorovski',1.0),('Spasovski',1.0),
    ('Ristovski',1.0),('Kostovski',1.0),('Ilievski',1.0),('Mitrovski',1.0),('Lazarevski',1.0),
    ('Markovski',1.0),('Stevanovski',1.0),('Bogdanovski',1.0),('Pavlovski',1.0),('Velkovski',1.0),
    ('Cvetkovski',1.0),('Krstevski',1.0),('Andonovski',1.0),('Nacevski',1.0),('Mihajlovski',1.0),
    ('Boshkovski',1.0),('Nestorovski',1.0),('Petkovski',1.0),('Ivanovski',1.0),('Simonovski',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_mk);

INSERT INTO names_me ("name", freq)
SELECT * FROM (VALUES
    ('Nikola',1.0),('Marko',1.0),('Luka',1.0),('Vuk',1.0),('Stefan',1.0),('Petar',1.0),
    ('Milos',1.0),('Andrija',1.0),('Bozidar',1.0),('Igor',1.0),('Vasilije',1.0),('Mihailo',1.0),
    ('Rade',1.0),('Dusan',1.0),('Sava',1.0),('Aleksa',1.0),('Boris',1.0),('Zoran',1.0),
    ('Dragan',1.0),('Nenad',1.0),('Veselin',1.0),('Radovan',1.0),('Ivan',1.0),('Filip',1.0),
    ('Bojan',1.0),('Milan',1.0),('Darko',1.0),('Goran',1.0),('Jovan',1.0),('Nemanja',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_me);

INSERT INTO lastnames_me (lastname, freq)
SELECT * FROM (VALUES
    ('Vukotic',1.0),('Djuranovic',1.0),('Markovic',1.0),('Radovic',1.0),('Mijovic',1.0),
    ('Kovacevic',1.0),('Petrovic',1.0),('Nikolic',1.0),('Bulatovic',1.0),('Gojkovic',1.0),
    ('Vujovic',1.0),('Stankovic',1.0),('Bozovic',1.0),('Lekic',1.0),('Pavlovic',1.0),
    ('Radonjic',1.0),('Mrvaljevic',1.0),('Ivanovic',1.0),('Jovanovic',1.0),('Rakocevic',1.0),
    ('Adzic',1.0),('Perovic',1.0),('Djukanovic',1.0),('Knezevic',1.0),('Vucinic',1.0),
    ('Lazarevic',1.0),('Simonovic',1.0),('Krivokapic',1.0),('Zivkovic',1.0),('Doderovic',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_me);

INSERT INTO names_xk ("name", freq)
SELECT * FROM (VALUES
    ('Arben',1.0),('Blerim',1.0),('Driton',1.0),('Fatmir',1.0),('Gezim',1.0),('Ilir',1.0),
    ('Jeton',1.0),('Kushtrim',1.0),('Labinot',1.0),('Mergim',1.0),('Naim',1.0),('Petrit',1.0),
    ('Qendrim',1.0),('Rinor',1.0),('Shpetim',1.0),('Trim',1.0),('Ujkan',1.0),('Valon',1.0),
    ('Visar',1.0),('Xhevdet',1.0),('Ylli',1.0),('Zymer',1.0),('Besnik',1.0),('Dardan',1.0),
    ('Endrit',1.0),('Flamur',1.0),('Genc',1.0),('Hekuran',1.0),('Ismail',1.0),('Luan',1.0)
) AS v("name", freq)
WHERE NOT EXISTS (SELECT 1 FROM names_xk);

INSERT INTO lastnames_xk (lastname, freq)
SELECT * FROM (VALUES
    ('Krasniqi',1.0),('Hoxha',1.0),('Berisha',1.0),('Gashi',1.0),('Kelmendi',1.0),
    ('Thaci',1.0),('Rrahmani',1.0),('Shala',1.0),('Zogaj',1.0),('Morina',1.0),
    ('Hasani',1.0),('Ademi',1.0),('Sylejmani',1.0),('Bytyqi',1.0),('Rama',1.0),
    ('Lushtaku',1.0),('Dauti',1.0),('Fazliu',1.0),('Mustafa',1.0),('Veseli',1.0),
    ('Kurteshi',1.0),('Jashari',1.0),('Pajaziti',1.0),('Rexhepi',1.0),('Sadiku',1.0),
    ('Bislimi',1.0),('Mullademi',1.0),('Ibrahimi',1.0),('Emini',1.0),('Ahmeti',1.0)
) AS v(lastname, freq)
WHERE NOT EXISTS (SELECT 1 FROM lastnames_xk);
