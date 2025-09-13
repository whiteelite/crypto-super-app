package entities

// Entity is a minimal marker interface used as a generic constraint
// and for embedding in domain structs across the codebase.
// Extend with common methods if needed (e.g., GetID()).
type Entity interface{}
