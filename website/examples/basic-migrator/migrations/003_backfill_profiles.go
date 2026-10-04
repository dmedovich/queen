package migrations

import (
	"context"
	"database/sql"

	"github.com/dmedovich/queen"
)

func Register003BackfillProfiles(q *queen.Queen) {
	q.MustAdd(queen.M{
		Version:        "003",
		Name:           "backfill_profiles",
		ManualChecksum: "backfill-profiles-v1",
		UpSQL: `
			CREATE TABLE profiles (
				user_id BIGINT PRIMARY KEY REFERENCES users(id),
				display_name TEXT NOT NULL
			);
		`,
		UpFunc:   up003BackfillProfiles,
		DownFunc: down003BackfillProfiles,
		DownSQL:  `DROP TABLE profiles;`,
	})
}

func up003BackfillProfiles(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO profiles (user_id, display_name)
		SELECT id, email FROM users
	`)
	return err
}

func down003BackfillProfiles(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM profiles`)
	return err
}
