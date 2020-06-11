package context

// ContextKey defines a dedicated type for keys to use on contexts
type Key string

// UserCRContextKey key under which the current User (from the database) is kept in the ctx
const UserCRContextKey Key = "user-cr"
