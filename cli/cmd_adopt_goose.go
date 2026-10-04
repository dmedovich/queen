package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/dmedovich/queen"
	"github.com/dmedovich/queen/drivers/postgres"
)

const defaultAdoptionLockTimeout = 30 * time.Minute

func (app *App) adoptGooseCmd() *cobra.Command {
	var gooseTable, expectedPlan string
	var apply, verifiedSchema bool
	cmd := &cobra.Command{
		Use: "adopt-goose", Short: "Preview or adopt existing Goose history into Queen on PostgreSQL",
		Long: "Preview is read-only and prints a plan fingerprint. Apply requires that fingerprint and confirmation that the database schema matches Goose's applied history.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if apply {
				if expectedPlan == "" || !verifiedSchema {
					return fmt.Errorf("--apply requires --plan from a prior preview and --verified-schema")
				}
			} else if expectedPlan != "" || verifiedSchema {
				return fmt.Errorf("--plan and --verified-schema are valid only with --apply")
			}
			return app.runWithQueen(cmd, func(ctx context.Context, q *queen.Queen) error {
				driver, ok := q.Driver().(*postgres.Driver)
				if !ok {
					return fmt.Errorf("adopt-goose currently supports PostgreSQL only")
				}
				sourceRef, err := quotePostgresTable(gooseTable)
				if err != nil {
					return err
				}
				targetRef, err := quotePostgresTable(driver.TableName)
				if err != nil {
					return err
				}
				registered := q.RegisteredMigrations()
				if !apply {
					plan, err := previewGooseAdoption(ctx, driver.DB, gooseTable, driver.TableName, sourceRef, targetRef, registered)
					if err != nil {
						return err
					}
					printGooseAdoptionPlan(plan)
					return nil
				}
				if err := app.checkConfirmation("adopt applied Goose migration history"); err != nil {
					return err
				}
				timeout := app.config.LockTimeout
				if timeout <= 0 {
					timeout = defaultAdoptionLockTimeout
				}
				plan, err := applyGooseAdoption(ctx, driver, gooseTable, driver.TableName, sourceRef, targetRef, registered, expectedPlan, timeout)
				if err != nil {
					return err
				}
				if plan.AlreadyDone {
					fmt.Println("Queen history already matches the Goose adoption plan")
				} else {
					fmt.Printf("Adopted %d Goose migration(s) into Queen history\n", len(plan.ToAdopt))
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&gooseTable, "goose-table", "goose_db_version", "Goose history table (table or schema.table)")
	cmd.Flags().BoolVar(&apply, "apply", false, "Apply a previously previewed adoption plan")
	cmd.Flags().StringVar(&expectedPlan, "plan", "", "Fingerprint printed by the preview")
	cmd.Flags().BoolVar(&verifiedSchema, "verified-schema", false, "Confirm the database schema matches Goose's applied migrations")
	return cmd
}

func previewGooseAdoption(ctx context.Context, db *sql.DB, sourceTable, targetTable, sourceRef, targetRef string, registered []queen.Migration) (gooseAdoptionPlan, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return gooseAdoptionPlan{}, err
	}
	defer func() { _ = tx.Rollback() }()
	plan, err := inspectGooseAdoption(ctx, tx, sourceTable, targetTable, sourceRef, targetRef, registered)
	if err != nil {
		return plan, err
	}
	if err := tx.Commit(); err != nil {
		return plan, err
	}
	return plan, nil
}

func applyGooseAdoption(ctx context.Context, driver *postgres.Driver, sourceTable, targetTable, sourceRef, targetRef string, registered []queen.Migration, expectedPlan string, timeout time.Duration) (plan gooseAdoptionPlan, retErr error) {
	if err := driver.Lock(ctx, timeout); err != nil {
		return plan, err
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		retErr = errors.Join(retErr, driver.Unlock(unlockCtx))
	}()
	if err := driver.Init(ctx); err != nil {
		return plan, err
	}
	err := driver.Exec(ctx, sql.LevelDefault, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("LOCK TABLE %s, %s IN SHARE ROW EXCLUSIVE MODE", sourceRef, targetRef)); err != nil {
			return fmt.Errorf("lock Goose and Queen history: %w", err)
		}
		var err error
		plan, err = inspectGooseAdoption(ctx, tx, sourceTable, targetTable, sourceRef, targetRef, registered)
		if err != nil {
			return err
		}
		if plan.Fingerprint != expectedPlan {
			return fmt.Errorf("adoption plan changed since preview; run adopt-goose again (current plan: %s)", plan.Fingerprint)
		}
		if plan.AlreadyDone {
			return nil
		}
		meta := &queen.MigrationMetadata{AppliedBy: currentUsername(), Action: "adopt_goose", Status: "success"}
		for i := range plan.ToAdopt {
			if err := driver.RecordTx(ctx, tx, &plan.ToAdopt[i], meta); err != nil {
				return fmt.Errorf("record Goose version %s: %w", plan.ToAdopt[i].Version, err)
			}
		}
		return nil
	})
	return plan, err
}
