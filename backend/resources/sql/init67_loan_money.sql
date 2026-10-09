-- Loan money (Economy Phase 4, "loan money" bullet): AcceptLoan used to move no money at all.
-- A lender can now attach three independent, all-optional economic terms to a loan listing - see
-- common/loans.go's CreateLoanListing/AcceptLoan and engine/wages.go:
--   * loan_fee: a one-time fee the borrower pays the lender when the loan is accepted, like a
--     mini transfer fee.
--   * wage_contribution_percent: the percentage of the player's wage the lender keeps covering
--     for the duration of the loan - the borrower only pays the remainder each matchday, and the
--     lender is charged the covered portion separately (see engine.deductWages).
--   * buy_option_fee: a fixed fee the borrower can pay at any time during the loan to make the
--     move permanent (see common.ExerciseLoanBuyOption) - the same money movement as a normal
--     transfer, just without moving the player (they already play for the borrower).
-- All three default to zero/off, so every loan already in flight when this ships behaves exactly
-- as before.
ALTER TABLE loan_listings ADD COLUMN IF NOT EXISTS loan_fee bigint NOT NULL DEFAULT 0;
ALTER TABLE loan_listings ADD COLUMN IF NOT EXISTS wage_contribution_percent smallint NOT NULL DEFAULT 0;
ALTER TABLE loan_listings ADD COLUMN IF NOT EXISTS buy_option_fee bigint NOT NULL DEFAULT 0;

-- Carried over from the listing at accept time (see common.AcceptLoan) - the listing row is
-- deleted once accepted, so the active loan needs its own copy of whatever terms applied.
ALTER TABLE player_loans ADD COLUMN IF NOT EXISTS wage_contribution_percent smallint NOT NULL DEFAULT 0;
ALTER TABLE player_loans ADD COLUMN IF NOT EXISTS buy_option_fee bigint NOT NULL DEFAULT 0;
