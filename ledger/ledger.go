package ledger

// Ledger tracks the state of every translatable unit in the codebase.
type Ledger interface {
	// Init creates the ledger schema (idempotent).
	Init() error

	// AddUnit adds a new translation unit. If it already exists (by ID), it is skipped.
	AddUnit(u *Unit) error

	// UpdateUnit updates an existing unit.
	UpdateUnit(u *Unit) error

	// GetUnit retrieves a unit by ID.
	GetUnit(id string) (*Unit, error)

	// NextUnit returns the next untranslated unit respecting tier order.
	// Returns nil, nil when there are no more units to translate.
	NextUnit() (*Unit, error)

	// ListUnits returns units, optionally filtered by status (empty = all).
	ListUnits(status Status) ([]*Unit, error)

	// Summary returns aggregate counts by status.
	Summary() (*Summary, error)

	// Commit creates a Dolt commit with the given message.
	Commit(msg string) error

	// Diff returns the Dolt diff of uncommitted changes.
	Diff() (string, error)

	// Log returns recent Dolt commit log.
	Log(n int) (string, error)

	// Close cleans up resources.
	Close() error
}
