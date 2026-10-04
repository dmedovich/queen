package importcmd

import (
	"fmt"
	"strings"
)

func gooseRegisterFuncName(version, name string) string {
	funcName := "Register" + version
	capitalizeNext := true
	for _, r := range name {
		if r == '_' || r == '-' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext && r >= 'a' && r <= 'z' {
			funcName += string(r - ('a' - 'A'))
			capitalizeNext = false
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			funcName += string(r)
			capitalizeNext = false
		}
	}
	return funcName
}

func generateQueenMigrationFile(version, name, funcName, upSQL, downSQL string, noTransaction bool) string {
	upSQL = strings.ReplaceAll(upSQL, "`", "` + \"`\" + `")
	downSQL = strings.ReplaceAll(downSQL, "`", "` + \"`\" + `")
	nonTransactionalField := ""
	if noTransaction {
		nonTransactionalField = "\n\t\tNonTransactional: true,"
	}

	return fmt.Sprintf(`package migrations

import (
	"github.com/dmedovich/queen"
)

func %s(q *queen.Queen) {
	q.MustAdd(queen.M{
		Version: "%s",
		Name:    "%s",%s
		UpSQL: `+"`"+`
%s
		`+"`"+`,
		DownSQL: `+"`"+`
%s
		`+"`"+`,
	})
}
`, funcName, version, name, nonTransactionalField, upSQL, downSQL)
}

func generateRegistrationFile(registrationCalls string) string {
	return fmt.Sprintf(`package migrations

import (
	"github.com/dmedovich/queen"
)

func Register(q *queen.Queen) {
%s
}
`, registrationCalls)
}
