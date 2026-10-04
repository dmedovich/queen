package migrations

import "github.com/dmedovich/queen"

func Register001CreateUsers(q *queen.Queen) {
	q.MustAdd(queen.M{
		Version: "001",
		Name:    "create_users",
		UpSQL: `
			CREATE TABLE users (
				id BIGSERIAL PRIMARY KEY,
				email TEXT NOT NULL UNIQUE,
				created_at TIMESTAMP NOT NULL DEFAULT NOW()
			);
		`,
		DownSQL: `DROP TABLE users;`,
	})
}
