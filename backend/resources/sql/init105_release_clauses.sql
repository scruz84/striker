-- Release clauses (openspec/specs/release-clauses): the optional, public fee any club may pay to
-- sign a player without his club's consent. NULL means no clause. release_clause_set_at is when
-- the contract that carries it was agreed (it can only be paid 14 days later). A transfer offer and
-- a pre-contract offer may propose one for the new contract. Re-runnable.
ALTER TABLE players ADD COLUMN IF NOT EXISTS release_clause bigint;
ALTER TABLE players ADD COLUMN IF NOT EXISTS release_clause_set_at timestamptz;
ALTER TABLE transfer_offers ADD COLUMN IF NOT EXISTS offered_release_clause bigint;
ALTER TABLE IF EXISTS pre_contract_offers ADD COLUMN IF NOT EXISTS release_clause bigint;
