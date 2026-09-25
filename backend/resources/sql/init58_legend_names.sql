-- Legend names for the random name generator (see common.GetPlayerName) - a small chance for a
-- newly generated player to be named after a football legend, or a legend-sounding name, instead
-- of drawing from the country's regular names_<suffix>/lastnames_<suffix> pool. Kept as a single
-- (first_name, last_name) pair per row, unlike names_<suffix>/lastnames_<suffix>'s independent
-- draws, so "Lionel" always lands with "Messi" rather than being mixed with an unrelated last
-- name.
-- Unique on the (first_name, last_name) pair so this file's own INSERT stays safe to re-run on
-- every startup (see common.applyPendingMigrations) without duplicating a row each time, and so
-- a future migration adding more legends can use the same ON CONFLICT DO NOTHING pattern.
CREATE TABLE IF NOT EXISTS legend_names
(
     first_name varchar(100) NOT NULL,
     last_name varchar(100) NOT NULL,
     UNIQUE (first_name, last_name)
);

INSERT INTO legend_names (first_name, last_name) VALUES
    ('Lionel', 'Messi'),
    ('Cristiano', 'Ronaldo'),
    ('Diego', 'Maradona'),
    ('Edson Arantes', 'do Nascimento'),
    ('Ronaldinho', 'Gaucho'),
    ('Zinedine', 'Zidane'),
    ('Ronaldo', 'Nazario'),
    ('Andres', 'Iniesta'),
    ('Xavi', 'Hernandez'),
    ('Ricardo', 'Kaka'),
    ('Thierry', 'Henry'),
    ('Roberto', 'Carlos'),
    ('Iker', 'Casillas'),
    ('Franz', 'Beckenbauer'),
    ('Johan', 'Cruyff'),
    ('George', 'Best'),
    ('Alfredo', 'Di Stefano'),
    ('Michel', 'Platini'),
    ('Marco', 'van Basten'),
    ('Paolo', 'Maldini'),
    ('Yuri', 'de Souza'),
    ('Kaito', 'Nishimura'),
    ('Erik', 'van der Berg'),
    ('Mateus', 'Ferreira Lima'),
    ('Aleksandr', 'Volkov'),
    ('Dimitri', 'Papadopoulos'),
    ('Chidi', 'Okonkwo'),
    ('Rashid', 'Al-Farsi'),
    ('Luca', 'Bellandi'),
    ('Nikolai', 'Petrov'),
    ('Sergio', 'Cruz'),
    ('Marcos', 'Martínez'),
    ('Javier', 'Sáez'),
    ('Fernando', 'Viforcos'),
    ('Alberto', 'Fernández'),
    ('Martín', 'Rodríguez'),
    ('Patrick', 'Mattiney'),
    ('Marcelo', 'Buelta')
ON CONFLICT (first_name, last_name) DO NOTHING;
