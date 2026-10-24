package imap

// StoreOptions contains options for the STORE command.
type StoreOptions struct {
	UnchangedSince    uint64 // requires CONDSTORE
	UnchangedSinceSet bool   // distinguishes UNCHANGEDSINCE 0
}

// StoreFlagsOp is a flag operation: set, add or delete.
type StoreFlagsOp int

const (
	StoreFlagsSet StoreFlagsOp = iota
	StoreFlagsAdd
	StoreFlagsDel
)

// StoreFlags alters message flags.
type StoreFlags struct {
	Op     StoreFlagsOp
	Silent bool
	Flags  []Flag
}
