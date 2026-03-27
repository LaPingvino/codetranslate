package ledger

import "time"

type Status string

const (
	StatusTodo       Status = "todo"
	StatusWIP        Status = "wip"
	StatusTranslated Status = "translated"
	StatusCompiles   Status = "compiles"
	StatusTested     Status = "tested"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
)

type Unit struct {
	ID          string
	SourceFile  string
	SourceName  string // function/type/const name
	SourceLang  string
	TargetFile  string
	TargetName  string
	TargetLang  string
	Kind        string // function, type, method, const, var, macro
	Status      Status
	Tier        int // dependency tier (lower = translate first)
	Model       string
	Attempts    int
	LastError   string
	SourceCode  string
	Translation string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Summary struct {
	Total      int
	Todo       int
	WIP        int
	Translated int
	Compiles   int
	Tested     int
	Done       int
	Failed     int
}
