-- Name pools for three more seeded countries (en/fr/de), mirroring the existing names_es/
-- lastnames_es schema. Curated common first/last names (~60 each) rather than a dump the size of
-- names_es/lastnames_es (~29k rows) - a deliberately smaller pool, disclosed as a scope cut: more
-- repetition across a large squad of teams than the Spanish pool has, but real, common names.
CREATE TABLE IF NOT EXISTS names_en ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_en (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_fr ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_fr (lastname varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS names_de ("name" varchar(100) NULL, freq float8 NULL);
CREATE TABLE IF NOT EXISTS lastnames_de (lastname varchar(100) NULL, freq float8 NULL);

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
