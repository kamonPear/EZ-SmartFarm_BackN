-- =============================================================================
-- 002_fix_egg_vaccine_health_coop_fk.sql
--
-- Forward migration that fixes a pre-existing bug (unrelated to the login/auth
-- feature in 001, just discovered while building it): egg.coop_id and
-- vaccine.coop_id were BIGINT while coop.coop_id is INT, so MySQL rejected the
-- foreign key every time the backend started (error 3780) and the FK was
-- silently missing. health.coop_id had the same latent BIGINT-vs-INT gap in its
-- Go model tag, but happened to already be INT in this database - fixed the
-- model anyway so a fresh database (or a future AutoMigrate) can't reintroduce
-- the same bug there.
--
-- This file is a plain-SQL mirror of what database/migration.go's MigrateModels
-- already does automatically on every backend startup. You normally do NOT need
-- to run this by hand - it exists for manual review/audit and as an emergency
-- fallback. Safe to re-run.
--
-- Take a backup first regardless - this is a live, in-use database.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. Narrow egg.coop_id and vaccine.coop_id from BIGINT back to INT, matching
--    coop.coop_id. (Skip this ALTER on either column if it's already INT.)
-- -----------------------------------------------------------------------------
ALTER TABLE `egg`     MODIFY COLUMN `coop_id` INT NOT NULL;
ALTER TABLE `vaccine` MODIFY COLUMN `coop_id` INT NOT NULL;

-- -----------------------------------------------------------------------------
-- 2. Add the foreign keys that couldn't be created before the type fix.
--    ON DELETE CASCADE matches the two other real FKs onto coop
--    (device_ibfk_1, health_ibfk_1) - deleting a coop removes its egg/vaccine
--    records along with it, same as it already does for devices/health.
-- -----------------------------------------------------------------------------
ALTER TABLE `egg`
  ADD CONSTRAINT `fk_coop_eggs` FOREIGN KEY (`coop_id`) REFERENCES `coop`(`coop_id`) ON DELETE CASCADE;

ALTER TABLE `vaccine`
  ADD CONSTRAINT `fk_coop_vaccines` FOREIGN KEY (`coop_id`) REFERENCES `coop`(`coop_id`) ON DELETE CASCADE;
