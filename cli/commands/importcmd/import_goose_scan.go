package importcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dmedovich/queen"
)

func scanGooseMigrations(sourcePath string, dryRun bool) ([]gooseMigration, error) {
	goFiles, err := filepath.Glob(filepath.Join(sourcePath, "*.go"))
	if err != nil {
		return nil, fmt.Errorf("failed to scan Go migrations: %w", err)
	}
	for _, file := range goFiles {
		if _, _, ok := parseGooseMigrationFilename(filepath.Base(file), ".go"); ok {
			return nil, fmt.Errorf("goose Go migration %s cannot be converted automatically", filepath.Base(file))
		}
	}
	files, err := filepath.Glob(filepath.Join(sourcePath, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to scan files: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no goose migration files found in %s", sourcePath)
	}
	fmt.Printf("Found %d migration file(s)\n\n", len(files))
	migrations := make([]gooseMigration, 0, len(files))
	seenVersions := make(map[int64]string, len(files))
	for _, file := range files {
		migration, err := readGooseMigration(file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(file), err)
		}
		numericVersion, _ := strconv.ParseInt(migration.version, 10, 64)
		if previous, exists := seenVersions[numericVersion]; exists {
			return nil, fmt.Errorf("duplicate goose version %d in %s and %s", numericVersion, previous, filepath.Base(file))
		}
		seenVersions[numericVersion] = filepath.Base(file)
		migrations = append(migrations, migration)
		fmt.Printf("  %s\n", filepath.Base(file))
		if dryRun {
			fmt.Printf("    Version: %s, Name: %s, NonTransactional: %t\n", migration.version, migration.name, migration.noTransaction)
		}
	}
	return migrations, nil
}

func readGooseMigration(file string) (gooseMigration, error) {
	version, name, ok := parseGooseFilename(filepath.Base(file))
	if !ok {
		return gooseMigration{}, fmt.Errorf("invalid goose migration filename (expected numeric_version_name.sql)")
	}
	content, err := os.ReadFile(file)
	if err != nil {
		return gooseMigration{}, fmt.Errorf("read migration: %w", err)
	}
	upSQL, downSQL, noTransaction, err := parseGooseSQL(string(content))
	if err != nil {
		return gooseMigration{}, err
	}
	if noTransaction && (strings.Count(upSQL, ";") > 1 || strings.Count(downSQL, ";") > 1) {
		return gooseMigration{}, fmt.Errorf("NO TRANSACTION migration contains multiple SQL statements; split it manually before import")
	}
	m := queen.M{Version: version, Name: name, UpSQL: upSQL, DownSQL: downSQL, NonTransactional: noTransaction}
	if err := m.Validate(); err != nil {
		return gooseMigration{}, fmt.Errorf("migration is invalid in Queen: %w", err)
	}
	return gooseMigration{file: file, version: version, name: name, upSQL: upSQL, downSQL: downSQL, noTransaction: noTransaction}, nil
}

func parseGooseFilename(basename string) (version, name string, ok bool) {
	return parseGooseMigrationFilename(basename, ".sql")
}

func parseGooseMigrationFilename(basename, extension string) (version, name string, ok bool) {
	if !strings.HasSuffix(basename, extension) {
		return "", "", false
	}
	parts := strings.SplitN(strings.TrimSuffix(basename, extension), "_", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	for _, r := range parts[0] {
		if r < '0' || r > '9' {
			return "", "", false
		}
	}
	versionID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || versionID <= 0 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func parseGooseSQL(content string) (upSQL, downSQL string, noTransaction bool, err error) {
	const (
		sectionNone = iota
		sectionUp
		sectionDown
	)
	section := sectionNone
	upCount, downCount := 0, 0
	inStatement := false
	var upLines, downLines []string
	for lineNumber, line := range strings.Split(content, "\n") {
		// Goose annotations must start at the beginning of a line.
		if strings.HasPrefix(line, "--") {
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[0] == "--" && strings.EqualFold(fields[1], "+goose") {
				annotation := strings.ToUpper(strings.Join(fields[2:], " "))
				switch annotation {
				case "UP":
					if upCount != 0 || section == sectionDown || inStatement {
						return "", "", false, fmt.Errorf("invalid or duplicate Up annotation on line %d", lineNumber+1)
					}
					upCount++
					section = sectionUp
				case "DOWN":
					if upCount != 1 || downCount != 0 || inStatement {
						return "", "", false, fmt.Errorf("invalid or duplicate Down annotation on line %d", lineNumber+1)
					}
					downCount++
					section = sectionDown
				case "NO TRANSACTION":
					if section != sectionNone || noTransaction {
						return "", "", false, fmt.Errorf("NO TRANSACTION must appear once before Up")
					}
					noTransaction = true
				case "STATEMENTBEGIN":
					if section == sectionNone || inStatement {
						return "", "", false, fmt.Errorf("invalid StatementBegin on line %d", lineNumber+1)
					}
					inStatement = true
				case "STATEMENTEND":
					if !inStatement {
						return "", "", false, fmt.Errorf("unmatched StatementEnd on line %d", lineNumber+1)
					}
					inStatement = false
				case "ENVSUB ON", "ENVSUB OFF":
					return "", "", false, fmt.Errorf("ENVSUB is not supported by the importer; substitute variables explicitly before import")
				default:
					return "", "", false, fmt.Errorf("unsupported goose annotation %q on line %d", annotation, lineNumber+1)
				}
				continue
			}
		}
		switch section {
		case sectionUp:
			upLines = append(upLines, line)
		case sectionDown:
			downLines = append(downLines, line)
		}
	}
	if upCount != 1 || inStatement {
		return "", "", false, fmt.Errorf("goose migration needs exactly one Up annotation and balanced statement markers")
	}
	upSQL = strings.TrimSpace(strings.Join(upLines, "\n"))
	downSQL = strings.TrimSpace(strings.Join(downLines, "\n"))
	if upSQL == "" {
		return "", "", false, fmt.Errorf("up section is empty")
	}
	if downCount != 0 && downSQL == "" {
		return "", "", false, fmt.Errorf("down section is empty")
	}
	return upSQL, downSQL, noTransaction, nil
}
