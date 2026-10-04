package main

import (
	"github.com/dmedovich/queen/cli"
	"github.com/dmedovich/queen/website/examples/basic-migrator/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cli.Run(migrations.Register)
}
