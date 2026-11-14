-- Ledger rows of 0 written by the cup round prize while the simulation never loaded a tie's round
-- (issue #398): every tie won through engine.SimulateCupTies paid CupPrizePerRound * 0 under
-- "Cup round 0 prize money". They moved no money and only clutter the finances page. The unpaid
-- prizes themselves are not paid back (5,000 per round, and which ties were already paid by the
-- default-result path cannot be told apart from the ledger).
--
-- Re-runnable: no new row of this shape is written any more (engine.awardCupRoundPrize skips a
-- prize of 0).

DELETE FROM team_transactions WHERE description = 'Cup round 0 prize money' AND amount = 0;
