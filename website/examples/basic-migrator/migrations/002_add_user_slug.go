package migrations

import "github.com/dmedovich/queen"

func Register002AddUserSlug(q *queen.Queen) {
	q.MustAdd(queen.M{
		Version: "002",
		Name:    "add_user_slug",
		UpSQL: `
			ALTER TABLE users ADD COLUMN slug TEXT;
			UPDATE users SET slug = LOWER(REPLACE(email, '@', '-'));
			ALTER TABLE users ALTER COLUMN slug SET NOT NULL;
			CREATE UNIQUE INDEX users_slug_idx ON users (slug);
		`,
		DownSQL: `
			DROP INDEX users_slug_idx;
			ALTER TABLE users DROP COLUMN slug;
		`,
	})
}
