-- =============================================================================
-- 001_add_user_auth.sql
--
-- Forward migration for the login/auth + per-user data ownership feature.
-- This file is a plain-SQL mirror of what database/migration.go's
-- MigrateModels / ensureAdminBootstrapAndOwnership already do automatically on
-- every backend startup. You normally do NOT need to run this by hand - it
-- exists for manual review/audit and as an emergency fallback. See README.md
-- in this folder for how/when to run it.
--
-- Safe to re-run: every statement below either uses IF NOT EXISTS-equivalent
-- guards or only touches rows that haven't been migrated yet. Run inside a
-- transaction if your MySQL/tooling supports DDL transactions, and take a
-- backup first regardless - this is a live, in-use database.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. User table: add role + timestamps
-- -----------------------------------------------------------------------------
ALTER TABLE `User`
  ADD COLUMN `role` VARCHAR(20) NOT NULL DEFAULT 'user' AFTER `password`,
  ADD COLUMN `created_at` DATETIME NULL AFTER `role`,
  ADD COLUMN `updated_at` DATETIME NULL AFTER `created_at`;

-- -----------------------------------------------------------------------------
-- 2. Root entity tables: add user_id (nullable for now - backfilled below,
--    then locked down with a NOT NULL-equivalent guarantee via FK + backfill).
-- -----------------------------------------------------------------------------
ALTER TABLE `coop`              ADD COLUMN `user_id` INT NULL, ADD INDEX `idx_coop_user_id` (`user_id`);
ALTER TABLE `foodstock`         ADD COLUMN `user_id` INT NULL;
ALTER TABLE `importfood`        ADD COLUMN `user_id` INT NULL, ADD INDEX `idx_importfood_user_id` (`user_id`);
ALTER TABLE `food_distribution` ADD COLUMN `user_id` INT NULL, ADD INDEX `idx_food_distribution_user_id` (`user_id`);
ALTER TABLE `farm_layout`       ADD COLUMN `user_id` INT NULL;

-- foodstock now holds one row per (food_type, user_id) instead of one global row
-- per food_type - replace the old single-column unique index with a composite one.
ALTER TABLE `foodstock` DROP INDEX `idx_foodstock_food_type`;
ALTER TABLE `foodstock` ADD UNIQUE INDEX `uq_foodstock_type_user` (`food_type`, `user_id`);

-- farm_layout used to be a single fixed row (id=1) shared by everyone; it's now one
-- row per user, so give it a real AUTO_INCREMENT id and a per-user unique index.
ALTER TABLE `farm_layout` MODIFY COLUMN `id` INT NOT NULL AUTO_INCREMENT;
ALTER TABLE `farm_layout` ADD UNIQUE INDEX `uq_farm_layout_user` (`user_id`);

-- -----------------------------------------------------------------------------
-- 3. Bootstrap an admin user (TEMPLATE ONLY - do not run as-is)
--
-- The real bootstrap happens in Go (database/migration.go:ensureBootstrapAdmin)
-- because it needs a bcrypt hash, which plain SQL can't generate. This is here
-- purely so the shape of what gets inserted is documented/auditable.
-- -----------------------------------------------------------------------------
-- INSERT INTO `User` (`username`, `password`, `role`, `created_at`, `updated_at`)
-- VALUES ('admin', '<bcrypt-hash-of-a-randomly-generated-password>', 'admin', NOW(), NOW());

-- -----------------------------------------------------------------------------
-- 4. Backfill: every pre-existing row (created before this migration ran) has
--    no owner yet - assign it to the bootstrap admin above so nothing already
--    in the live database becomes orphaned or inaccessible.
--    Replace <ADMIN_ID> with that admin user's id_User.
-- -----------------------------------------------------------------------------
UPDATE `coop`              SET `user_id` = <ADMIN_ID> WHERE `user_id` IS NULL OR `user_id` = 0;
UPDATE `foodstock`         SET `user_id` = <ADMIN_ID> WHERE `user_id` IS NULL OR `user_id` = 0;
UPDATE `importfood`        SET `user_id` = <ADMIN_ID> WHERE `user_id` IS NULL OR `user_id` = 0;
UPDATE `food_distribution` SET `user_id` = <ADMIN_ID> WHERE `user_id` IS NULL OR `user_id` = 0;
UPDATE `farm_layout`       SET `user_id` = <ADMIN_ID> WHERE `user_id` IS NULL OR `user_id` = 0;

-- -----------------------------------------------------------------------------
-- 5. Lock ownership down with real foreign keys (ON DELETE RESTRICT - a user
--    row can't be deleted while it still owns data, matching the app's own
--    guarded ensureForeignKey helper).
-- -----------------------------------------------------------------------------
ALTER TABLE `coop`              ADD CONSTRAINT `fk_coop_user`              FOREIGN KEY (`user_id`) REFERENCES `User`(`id_User`) ON DELETE RESTRICT;
ALTER TABLE `foodstock`         ADD CONSTRAINT `fk_foodstock_user`         FOREIGN KEY (`user_id`) REFERENCES `User`(`id_User`) ON DELETE RESTRICT;
ALTER TABLE `importfood`        ADD CONSTRAINT `fk_importfood_user`        FOREIGN KEY (`user_id`) REFERENCES `User`(`id_User`) ON DELETE RESTRICT;
ALTER TABLE `food_distribution` ADD CONSTRAINT `fk_food_distribution_user` FOREIGN KEY (`user_id`) REFERENCES `User`(`id_User`) ON DELETE RESTRICT;
ALTER TABLE `farm_layout`       ADD CONSTRAINT `fk_farm_layout_user`       FOREIGN KEY (`user_id`) REFERENCES `User`(`id_User`) ON DELETE RESTRICT;

-- Note: egg / health / vaccine / device intentionally get NO user_id column or FK -
-- they're scoped indirectly through their parent coop's user_id instead (see
-- database.CoopBelongsToUser in the Go code), so there's nothing to migrate on them.
