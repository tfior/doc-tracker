// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"context"
	"database/sql"
	"fmt"
)

// DBTX is the subset of *sql.DB and *sql.Tx that stores use to run SQL.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type txKey struct{}

// WithTx runs fn inside a database transaction. The transaction travels in the
// context passed to fn, so every store that uses Conn on that context joins it.
// If fn returns an error the transaction is rolled back; otherwise it is
// committed. A WithTx call inside another joins the outer transaction, so a
// unit of work can be composed from services that each use WithTx.
func WithTx(ctx context.Context, db *sql.DB, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return fn(ctx)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// Conn returns the transaction carried by ctx, or db when there is none.
// Stores call it for every query so they never begin or commit transactions
// themselves.
func Conn(ctx context.Context, db *sql.DB) DBTX {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return db
}

// RequireOneRow returns notFound when a statement affected no rows.
func RequireOneRow(result sql.Result, notFound error) error {
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return notFound
	}
	return nil
}
