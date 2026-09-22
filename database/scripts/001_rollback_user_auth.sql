-- =============================================================================
-- 001_rollback_user_auth.sql
--
-- Exact reverse of 001_add_user_auth.sql - undoes the DATABASE SCHEMA side of
-- the login/auth + per-user data ownership feature only.
--
-- IMPORTANT: this does NOT fully roll back the feature by itself. It only
-- undoes the schema changes below; the Go code (models/handlers/routes/
-- middleware on the feature/user-auth branch) also needs to be reverted
-- separately, e.g. `git checkout main` in the backend repo, otherwise the
-- running server's Go code will immediately try to read/write columns this
-- script just dropped and error out. Run this script only as part of a full
-- rollback that also reverts the code, not on its own.
--
-- This script deliberately does NOT drop the `User` table and does NOT delete
-- the bootstrap admin (or any other user) row - the original username/password
-- login data that predates this feature is left completely intact, only the
-- additive schema changes are undone.
--
-- Take a backup first regardless - this is a live, in-use database.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. Drop the foreign keys added in step 5 of the forward migration
-- -----------------------------------------------------------------------------
ALTER TABLE `coop`              DROP FOREIGN KEY `fk_coop_user`;
ALTER TABLE `foodstock`         DROP FOREIGN KEY `fk_foodstock_user`;
ALTER TABLE `importfood`        DROP FOREIGN KEY `fk_importfood_user`;
ALTER TABLE `food_distribution` DROP FOREIGN KEY `fk_food_distribution_user`;
ALTER TABLE `farm_layout`       DROP FOREIGN KEY `fk_farm_layout_user`;

-- -----------------------------------------------------------------------------
-- 2. Drop the per-user indexes/constraints added on farm_layout and foodstock
-- -----------------------------------------------------------------------------
ALTER TABLE `farm_layout` DROP INDEX `uq_farm_layout_user`;
ALTER TABLE `foodstock`   DROP INDEX `uq_foodstock_type_user`;

-- Restore the original single-column unique index the forward script dropped.
-- NOTE: this will fail if more than one row per food_type exists (i.e. more than
-- one user has imported that food type) - consolidate/delete extra rows first.
ALTER TABLE `foodstock` ADD UNIQUE INDEX `idx_foodstock_food_type` (`food_type`);

-- farm_layout.id: AUTO_INCREMENT is harmless to leave in place, but if you want
-- an exact mirror of the pre-migration schema, drop it (requires the table to
-- have at most one row again, since id was previously a fixed singleton key):
-- ALTER TABLE `farm_layout` MODIFY COLUMN `id` INT NOT NULL;

-- -----------------------------------------------------------------------------
-- 3. Drop the user_id columns (and their plain indexes) from the five root
--    entity tables
-- -----------------------------------------------------------------------------
ALTER TABLE `coop`              DROP INDEX `idx_coop_user_id`,              DROP COLUMN `user_id`;
ALTER TABLE `foodstock`                                                    DROP COLUMN `user_id`;
ALTER TABLE `importfood`        DROP INDEX `idx_importfood_user_id`,        DROP COLUMN `user_id`;
ALTER TABLE `food_distribution` DROP INDEX `idx_food_distribution_user_id`, DROP COLUMN `user_id`;
ALTER TABLE `farm_layout`                                                  DROP COLUMN `user_id`;

-- -----------------------------------------------------------------------------
-- 4. Drop role/created_at/updated_at from User. The table itself, and every
--    existing row (including the bootstrap admin), is left untouched.
-- -----------------------------------------------------------------------------
ALTER TABLE `User`
  DROP COLUMN `role`,
  DROP COLUMN `created_at`,
  DROP COLUMN `updated_at`;
