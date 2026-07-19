package textunmarshaler

// Simulates a Router that references generated wrappers that are missing or
// stale — packages.Load reports errors, but TypedID types remain resolvable.
var _ = missingGeneratedWrapper
