package agentserver

// Postgres browse layer: list databases, list tables, read rows.
// Read-only. Connection params arrive per-request; pgx pools are cached by DSN
// so repeated calls reuse connections. Add new browse features as functions
// here + a handler in server.go — nothing else needs to change.

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgConnReq carries connection params for a Postgres request.
type PgConnReq struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DB       string `json:"db"`
	SSLMode  string `json:"sslMode"`
}

// PgRowsReq selects a page of rows from one table.
type PgRowsReq struct {
	PgConnReq
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// PgTable is one user table.
type PgTable struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Rows   int64  `json:"rows"` // planner estimate (pg_class.reltuples)
}

// PgPage is one page of table rows. NULL cells are JSON null (nil pointer).
// Ctids[i] is the system row id for Rows[i], used to target deletes.
type PgPage struct {
	Columns []string    `json:"columns"`
	Types   []string    `json:"types"`
	Rows    [][]*string `json:"rows"`
	Ctids   []string    `json:"ctids"`
	HasMore bool        `json:"hasMore"`
	Offset  int         `json:"offset"`
	Limit   int         `json:"limit"`
}

// PgDeleteReq targets one row for deletion by its ctid.
type PgDeleteReq struct {
	PgConnReq
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Ctid   string `json:"ctid"`
}

// PgInsertReq inserts one row. Values maps column name -> value (nil = NULL);
// omitted columns fall back to their DB default.
type PgInsertReq struct {
	PgConnReq
	Schema string             `json:"schema"`
	Table  string             `json:"table"`
	Values map[string]*string `json:"values"`
}

// PgUpdateReq sets one column of one row (by ctid). Value nil = NULL.
type PgUpdateReq struct {
	PgConnReq
	Schema string  `json:"schema"`
	Table  string  `json:"table"`
	Ctid   string  `json:"ctid"`
	Column string  `json:"column"`
	Value  *string `json:"value"`
}

func (r PgConnReq) withDefaults() PgConnReq {
	if r.Host == "" {
		r.Host = "127.0.0.1"
	}
	if r.Port == 0 {
		r.Port = 5432
	}
	if r.User == "" {
		r.User = "postgres"
	}
	if r.SSLMode == "" {
		r.SSLMode = "disable"
	}
	return r
}

func (r PgConnReq) dsn(db string) string {
	if db == "" {
		db = "postgres" // bootstrap db for listing databases
	}
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(r.User, r.Password),
		Host:     fmt.Sprintf("%s:%d", r.Host, r.Port),
		Path:     "/" + db,
		RawQuery: "sslmode=" + r.SSLMode,
	}
	return u.String()
}

var (
	pgMu    sync.Mutex
	pgPools = map[string]*pgxpool.Pool{}
)

// pgPool returns a cached pool for the DSN, creating it on first use.
func pgPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pgMu.Lock()
	defer pgMu.Unlock()
	if p, ok := pgPools[dsn]; ok {
		return p, nil
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 4
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pgPools[dsn] = p
	return p, nil
}

// PgListDatabases returns non-template database names.
func PgListDatabases(ctx context.Context, req PgConnReq) ([]string, error) {
	req = req.withDefaults()
	pool, err := pgPool(ctx, req.dsn(req.DB))
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// PgListTables lists ordinary + partitioned tables in non-system schemas.
func PgListTables(ctx context.Context, req PgConnReq) ([]PgTable, error) {
	req = req.withDefaults()
	if req.DB == "" {
		return nil, fmt.Errorf("db required")
	}
	pool, err := pgPool(ctx, req.dsn(req.DB))
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT n.nspname, c.relname, c.reltuples::bigint
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r','p')
		  AND n.nspname NOT IN ('pg_catalog','information_schema')
		ORDER BY n.nspname, c.relname`)
	if err != nil {
		return nil, err
	}

	out := []PgTable{}
	for rows.Next() {
		var t PgTable
		if err := rows.Scan(&t.Schema, &t.Name, &t.Rows); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// reltuples is a planner estimate and is 0/-1 for never-analyzed tables.
	// For those, run a bounded exact count so the UI doesn't show a wrong 0.
	for i := range out {
		if out[i].Rows > 0 {
			continue
		}
		out[i].Rows = 0
		cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		ident := pgx.Identifier{out[i].Schema, out[i].Name}.Sanitize()
		var n int64
		if err := pool.QueryRow(cctx, "SELECT count(*) FROM "+ident).Scan(&n); err == nil {
			out[i].Rows = n
		}
		cancel()
	}
	return out, nil
}

// PgQueryRows returns a page of rows from schema.table.
// Uses limit+1 fetch to set HasMore without a count(*).
func PgQueryRows(ctx context.Context, req PgRowsReq) (PgPage, error) {
	base := req.PgConnReq.withDefaults()
	if base.DB == "" || req.Table == "" {
		return PgPage{}, fmt.Errorf("db and table required")
	}
	schema := req.Schema
	if schema == "" {
		schema = "public"
	}
	limit := req.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	pool, err := pgPool(ctx, base.dsn(base.DB))
	if err != nil {
		return PgPage{}, err
	}

	// Identifier-safe: Sanitize() quotes schema/table, blocking injection.
	// Inject ctid as the first column so rows can be targeted for deletion.
	ident := pgx.Identifier{schema, req.Table}.Sanitize()
	sql := fmt.Sprintf("SELECT _pgm.ctid::text, _pgm.* FROM %s AS _pgm LIMIT $1 OFFSET $2", ident)
	rows, err := pool.Query(ctx, sql, limit+1, offset)
	if err != nil {
		return PgPage{}, err
	}
	defer rows.Close()

	// fds[0] is the injected ctid; user columns start at index 1.
	fds := rows.FieldDescriptions()
	tm := pgtype.NewMap()
	cols := make([]string, len(fds)-1)
	types := make([]string, len(fds)-1)
	for i := 1; i < len(fds); i++ {
		cols[i-1] = fds[i].Name
		if t, ok := tm.TypeForOID(fds[i].DataTypeOID); ok {
			types[i-1] = t.Name
		} else {
			types[i-1] = fmt.Sprintf("oid:%d", fds[i].DataTypeOID)
		}
	}

	page := PgPage{Columns: cols, Types: types, Rows: [][]*string{}, Ctids: []string{}, Limit: limit, Offset: offset}
	for rows.Next() {
		if len(page.Rows) >= limit {
			page.HasMore = true
			break
		}
		vals, err := rows.Values()
		if err != nil {
			return PgPage{}, err
		}
		ctid := ""
		if len(vals) > 0 {
			if s := pgValToStr(vals[0]); s != nil {
				ctid = *s
			}
		}
		cells := make([]*string, 0, len(vals)-1)
		for i := 1; i < len(vals); i++ {
			cells = append(cells, pgValToStr(vals[i]))
		}
		page.Rows = append(page.Rows, cells)
		page.Ctids = append(page.Ctids, ctid)
	}
	return page, rows.Err()
}

// pgValToStr renders a cell value as a string, or nil for SQL NULL.
func pgValToStr(v any) *string {
	if v == nil {
		return nil
	}
	var s string
	switch x := v.(type) {
	case string:
		s = x
	case []byte:
		s = string(x)
	case time.Time:
		s = x.Format(time.RFC3339)
	case fmt.Stringer:
		s = x.String()
	default:
		s = fmt.Sprintf("%v", x)
	}
	return &s
}

// PgDeleteRow deletes one row identified by ctid. Returns rows affected.
func PgDeleteRow(ctx context.Context, req PgDeleteReq) (int64, error) {
	base := req.PgConnReq.withDefaults()
	if base.DB == "" || req.Table == "" || req.Ctid == "" {
		return 0, fmt.Errorf("db, table and ctid required")
	}
	schema := req.Schema
	if schema == "" {
		schema = "public"
	}
	pool, err := pgPool(ctx, base.dsn(base.DB))
	if err != nil {
		return 0, err
	}
	ident := pgx.Identifier{schema, req.Table}.Sanitize()
	tag, err := pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE ctid = $1::tid", ident), req.Ctid)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// PgInsertRow inserts one row from the provided column/value map. Values are
// sent as text and cast by Postgres to the column type. Returns rows affected.
func PgInsertRow(ctx context.Context, req PgInsertReq) (int64, error) {
	base := req.PgConnReq.withDefaults()
	if base.DB == "" || req.Table == "" || len(req.Values) == 0 {
		return 0, fmt.Errorf("db, table and at least one value required")
	}
	schema := req.Schema
	if schema == "" {
		schema = "public"
	}
	pool, err := pgPool(ctx, base.dsn(base.DB))
	if err != nil {
		return 0, err
	}
	cols := make([]string, 0, len(req.Values))
	placeholders := make([]string, 0, len(req.Values))
	args := make([]any, 0, len(req.Values))
	i := 1
	for col, val := range req.Values {
		cols = append(cols, pgx.Identifier{col}.Sanitize())
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		i++
		if val == nil {
			args = append(args, nil)
		} else {
			args = append(args, *val)
		}
	}
	ident := pgx.Identifier{schema, req.Table}.Sanitize()
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", ident, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	tag, err := pool.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// PgUpdateCell sets one column of the row identified by ctid. The value is sent
// as text and cast by Postgres to the column type. Returns rows affected.
func PgUpdateCell(ctx context.Context, req PgUpdateReq) (int64, error) {
	base := req.PgConnReq.withDefaults()
	if base.DB == "" || req.Table == "" || req.Ctid == "" || req.Column == "" {
		return 0, fmt.Errorf("db, table, ctid and column required")
	}
	schema := req.Schema
	if schema == "" {
		schema = "public"
	}
	pool, err := pgPool(ctx, base.dsn(base.DB))
	if err != nil {
		return 0, err
	}
	ident := pgx.Identifier{schema, req.Table}.Sanitize()
	col := pgx.Identifier{req.Column}.Sanitize()
	sql := fmt.Sprintf("UPDATE %s SET %s = $1 WHERE ctid = $2::tid", ident, col)
	tag, err := pool.Exec(ctx, sql, req.Value, req.Ctid)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
