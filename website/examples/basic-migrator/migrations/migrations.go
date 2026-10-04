package migrations

import "github.com/dmedovich/queen"

func Register(q *queen.Queen) {
	Register001CreateUsers(q)
	Register002AddUserSlug(q)
	Register003BackfillProfiles(q)
}
