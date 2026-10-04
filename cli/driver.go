package cli

import (
	"database/sql"
	"fmt"

	"github.com/dmedovich/queen"
	"github.com/dmedovich/queen/drivers/clickhouse"
	"github.com/dmedovich/queen/drivers/cockroachdb"
	"github.com/dmedovich/queen/drivers/mssql"
	"github.com/dmedovich/queen/drivers/mysql"
	"github.com/dmedovich/queen/drivers/postgres"
	"github.com/dmedovich/queen/drivers/sqlite"
)

const (
	DriverPostgres   = "postgres"
	DriverPostgreSQL = "postgresql"
	DriverMySQL      = "mysql"
	DriverSQLite     = "sqlite"
	DriverSQLite3    = "sqlite3"
	DriverClickHouse = "clickhouse"
	DriverCockroach  = "cockroachdb"
	DriverMSSQL      = "mssql"
	DriverSQLServer  = "sqlserver"

	SQLDriverPostgres   = "pgx"
	SQLDriverMySQL      = "mysql"
	SQLDriverSQLite     = "sqlite3"
	SQLDriverClickHouse = "clickhouse"
	SQLDriverMSSQL      = "sqlserver"
)

func getSQLDriverName(driverName string) string {
	switch driverName {
	case DriverPostgres, DriverPostgreSQL:
		return SQLDriverPostgres
	case DriverMySQL:
		return SQLDriverMySQL
	case DriverSQLite, DriverSQLite3:
		return SQLDriverSQLite
	case DriverClickHouse:
		return SQLDriverClickHouse
	case DriverCockroach:
		return SQLDriverPostgres
	case DriverMSSQL, DriverSQLServer:
		return SQLDriverMSSQL
	default:
		return driverName
	}
}

func (app *App) createDriver(db *sql.DB) (queen.Driver, error) {
	tableName := app.config.Table
	if tableName == "" {
		tableName = DefaultTableName
	}

	switch app.config.Driver {
	case DriverPostgres, DriverPostgreSQL, "pgx":
		return postgres.NewWithTableName(db, tableName), nil

	case "mysql":
		return mysql.NewWithTableName(db, tableName), nil

	case "sqlite", "sqlite3":
		return sqlite.NewWithTableName(db, tableName), nil

	case "clickhouse":
		return clickhouse.NewWithTableName(db, tableName)

	case DriverCockroach:
		return cockroachdb.NewWithTableName(db, tableName)

	case DriverMSSQL, DriverSQLServer:
		return mssql.NewWithTableName(db, tableName), nil

	default:
		return nil, fmt.Errorf(
			"unsupported driver: %s (supported: postgres, mysql, sqlite, clickhouse, cockroachdb, mssql)",
			app.config.Driver,
		)
	}
}
