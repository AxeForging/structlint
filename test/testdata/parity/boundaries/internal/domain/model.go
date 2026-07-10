package domain

// Violates the boundary: domain imports db.
import _ "example.com/fixture/internal/db"

type Entity struct{ ID string }
