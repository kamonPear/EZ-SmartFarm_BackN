# Migration scripts (login/auth feature)

Each numbered pair is a plain-SQL mirror of schema changes `database/migration.go`
(`MigrateModels`) already applies automatically, idempotently, every time the backend
starts up - normal operation does **not** require running any of these files by hand.
They exist for manual review/audit of exactly what changed on the live database, and as
an emergency fallback (e.g. to inspect or apply a change outside of running the Go
binary, or to manually undo a schema change if a branch needs to be backed out). To run
one by hand: `mysql -h <host> -P <port> -u <user> -p <db> < 00N_rollback_....sql` (swap
in the `00N_add_...sql`/`00N_fix_...sql` file for the forward direction), after taking a
backup - this project points at a live, already-in-use remote MySQL instance, not a
throwaway dev database.

- **001** - the login/auth feature itself: `User.role`/timestamps (role was later
  dropped again, see the git history), `user_id` on the five per-user root tables, the
  bootstrap admin account.
- **002** - unrelated pre-existing bug found while building 001: `egg.coop_id` and
  `vaccine.coop_id` were BIGINT while `coop.coop_id` is INT, so their foreign keys to
  `coop` silently failed to get created on every startup. Narrows the columns back to
  INT and adds the missing FKs.
