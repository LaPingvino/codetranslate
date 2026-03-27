package ledger

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const schema = `CREATE TABLE IF NOT EXISTS translation_units (
	id VARCHAR(64) PRIMARY KEY,
	source_file VARCHAR(1024),
	source_name VARCHAR(512),
	source_lang VARCHAR(64),
	target_file VARCHAR(1024),
	target_name VARCHAR(512),
	target_lang VARCHAR(64),
	kind VARCHAR(64),
	status VARCHAR(32) DEFAULT 'todo',
	tier INT DEFAULT 0,
	model VARCHAR(128) DEFAULT '',
	attempts INT DEFAULT 0,
	last_error TEXT,
	source_code LONGTEXT,
	translation LONGTEXT
);`

// DoltLedger implements Ledger using the Dolt CLI.
type DoltLedger struct {
	dir string // path to the Dolt repo
}

// NewDolt creates a new DoltLedger at the given directory.
func NewDolt(dir string) *DoltLedger {
	return &DoltLedger{dir: dir}
}

func (d *DoltLedger) Init() error {
	// Create directory if needed
	if err := os.MkdirAll(d.dir, 0755); err != nil {
		return fmt.Errorf("creating ledger dir: %w", err)
	}

	// Check if already initialized
	dotDolt := filepath.Join(d.dir, ".dolt")
	if _, err := os.Stat(dotDolt); os.IsNotExist(err) {
		if _, err := d.exec("init"); err != nil {
			return fmt.Errorf("dolt init: %w", err)
		}
	}

	// Create table
	if _, err := d.sql(schema); err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}

	// Initial commit if there are staged changes
	_, _ = d.exec("add", "-A")
	_, _ = d.exec("commit", "-m", "Initialize translation ledger")
	return nil
}

func (d *DoltLedger) AddUnit(u *Unit) error {
	u.ID = unitID(u)
	q := fmt.Sprintf(
		`INSERT IGNORE INTO translation_units (id, source_file, source_name, source_lang, target_file, target_name, target_lang, kind, status, tier, source_code)
		 VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s)`,
		quote(u.ID), quote(u.SourceFile), quote(u.SourceName), quote(u.SourceLang),
		quote(u.TargetFile), quote(u.TargetName), quote(u.TargetLang),
		quote(u.Kind), quote(string(u.Status)), u.Tier, quote(u.SourceCode),
	)
	_, err := d.sql(q)
	return err
}

func (d *DoltLedger) UpdateUnit(u *Unit) error {
	q := fmt.Sprintf(
		`UPDATE translation_units SET
			status = %s, model = %s, attempts = %d, last_error = %s,
			translation = %s, target_file = %s, target_name = %s
		 WHERE id = %s`,
		quote(string(u.Status)), quote(u.Model), u.Attempts, quote(u.LastError),
		quote(u.Translation), quote(u.TargetFile), quote(u.TargetName),
		quote(u.ID),
	)
	_, err := d.sql(q)
	return err
}

func (d *DoltLedger) GetUnit(id string) (*Unit, error) {
	rows, err := d.query(fmt.Sprintf("SELECT * FROM translation_units WHERE id = %s", quote(id)))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("unit %s not found", id)
	}
	return rowToUnit(rows[0]), nil
}

func (d *DoltLedger) NextUnit() (*Unit, error) {
	rows, err := d.query(
		`SELECT * FROM translation_units
		 WHERE status IN ('todo', 'failed')
		 ORDER BY tier ASC, source_file ASC, source_name ASC
		 LIMIT 1`,
	)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil // done
	}
	return rowToUnit(rows[0]), nil
}

func (d *DoltLedger) ListUnits(status Status) ([]*Unit, error) {
	q := "SELECT * FROM translation_units"
	if status != "" {
		q += fmt.Sprintf(" WHERE status = %s", quote(string(status)))
	}
	q += " ORDER BY tier ASC, source_file ASC, source_name ASC"
	rows, err := d.query(q)
	if err != nil {
		return nil, err
	}
	units := make([]*Unit, len(rows))
	for i, r := range rows {
		units[i] = rowToUnit(r)
	}
	return units, nil
}

func (d *DoltLedger) Summary() (*Summary, error) {
	rows, err := d.query("SELECT status, COUNT(*) as cnt FROM translation_units GROUP BY status")
	if err != nil {
		return nil, err
	}
	s := &Summary{}
	for _, r := range rows {
		cnt := jsonInt(r["cnt"])
		switch Status(jsonString(r["status"])) {
		case StatusTodo:
			s.Todo = cnt
		case StatusWIP:
			s.WIP = cnt
		case StatusTranslated:
			s.Translated = cnt
		case StatusCompiles:
			s.Compiles = cnt
		case StatusTested:
			s.Tested = cnt
		case StatusDone:
			s.Done = cnt
		case StatusFailed:
			s.Failed = cnt
		}
		s.Total += cnt
	}
	return s, nil
}

func (d *DoltLedger) Commit(msg string) error {
	if _, err := d.exec("add", "-A"); err != nil {
		return err
	}
	_, err := d.exec("commit", "-m", msg)
	return err
}

func (d *DoltLedger) Diff() (string, error) {
	return d.exec("diff")
}

func (d *DoltLedger) Log(n int) (string, error) {
	return d.exec("log", "-n", fmt.Sprintf("%d", n))
}

func (d *DoltLedger) Close() error {
	return nil
}

// exec runs a dolt command in the ledger directory.
func (d *DoltLedger) exec(args ...string) (string, error) {
	cmd := exec.Command("dolt", args...)
	cmd.Dir = d.dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// sql executes a SQL statement via dolt sql.
func (d *DoltLedger) sql(query string) (string, error) {
	return d.exec("sql", "-q", query)
}

// query executes a SQL query and returns rows as maps.
func (d *DoltLedger) query(q string) ([]map[string]interface{}, error) {
	out, err := d.exec("sql", "-r", "json", "-q", q)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w\n%s", err, out)
	}
	if out == "" {
		return nil, nil
	}
	var result struct {
		Rows []map[string]interface{} `json:"rows"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w\nraw: %s", err, out)
	}
	return result.Rows, nil
}

// unitID generates a stable ID from source file + name + kind.
func unitID(u *Unit) string {
	h := sha256.Sum256([]byte(u.SourceFile + ":" + u.SourceName + ":" + u.Kind))
	return fmt.Sprintf("%x", h[:8])
}

func rowToUnit(r map[string]interface{}) *Unit {
	return &Unit{
		ID:          jsonString(r["id"]),
		SourceFile:  jsonString(r["source_file"]),
		SourceName:  jsonString(r["source_name"]),
		SourceLang:  jsonString(r["source_lang"]),
		TargetFile:  jsonString(r["target_file"]),
		TargetName:  jsonString(r["target_name"]),
		TargetLang:  jsonString(r["target_lang"]),
		Kind:        jsonString(r["kind"]),
		Status:      Status(jsonString(r["status"])),
		Tier:        jsonInt(r["tier"]),
		Model:       jsonString(r["model"]),
		Attempts:    jsonInt(r["attempts"]),
		LastError:   jsonString(r["last_error"]),
		SourceCode:  jsonString(r["source_code"]),
		Translation: jsonString(r["translation"]),
		CreatedAt:   time.Time{},
		UpdatedAt:   time.Time{},
	}
}

func jsonString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func jsonInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		var i int
		fmt.Sscanf(n, "%d", &i)
		return i
	}
	return 0
}

// quote escapes a string for SQL. This is basic — sufficient for CLI-based dolt sql
// since we're not exposed to user input injection (we control all values).
func quote(s string) string {
	s = strings.ReplaceAll(s, "'", "''")
	s = strings.ReplaceAll(s, "\\", "\\\\")
	return "'" + s + "'"
}
