-- =============================================================================
-- 002_rollback_egg_vaccine_health_coop_fk.sql
--
-- Reverse of 002_fix_egg_vaccine_health_coop_fk.sql. Drops the two foreign keys
-- that migration added. Deliberately does NOT widen egg.coop_id/vaccine.coop_id
-- back to BIGINT - that was a bug, not a design choice, so there is nothing
-- correct to restore by reintroducing it. This only removes the FK constraints.
--
-- Reverting the Go code (`git checkout main` in the backend repo, or whichever
-- commit predates this fix) is also required for a full rollback - otherwise
-- the running server's migration will just recreate these FKs on its next boot.
--
-- Take a backup first regardless - this is a live, in-use database.
-- =============================================================================

ALTER TABLE `egg`     DROP FOREIGN KEY `fk_coop_eggs`;
ALTER TABLE `vaccine` DROP FOREIGN KEY `fk_coop_vaccines`;
