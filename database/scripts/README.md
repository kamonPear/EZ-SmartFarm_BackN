# Migration scripts (login/auth feature)

`001_add_user_auth.sql` and `001_rollback_user_auth.sql` are plain-SQL mirrors of the
schema changes the Go code in `database/migration.go` (`MigrateModels` /
`ensureAdminBootstrapAndOwnership`) already applies automatically, idempotently, every
time the backend starts up - normal operation does **not** require running either file
by hand. They exist for manual review/audit of exactly what the auth migration does to
the live database, and as an emergency fallback (e.g. to inspect or apply the change
outside of running the Go binary, or to manually undo the schema if the
`feature/user-auth` branch needs to be backed out). To run one by hand: `mysql -h <host>
-P <port> -u <user> -p <db> < 001_rollback_user_auth.sql` (swap in
`001_add_user_auth.sql` for the forward direction), after taking a backup - this project
points at a live, already-in-use remote MySQL instance, not a throwaway dev database.
