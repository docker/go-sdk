// Package legacyadapters provides conversion utilities to bridge between
// the modern Docker Go SDK types and legacy Docker CLI/Docker Engine API types.
//
// Deprecated: This entire module is deprecated and temporary. It will be removed
// in a future release when all Docker products have migrated to use the go-sdk
// natively. We strongly recommend avoiding this module in new projects and using
// the native go-sdk types directly instead.
//
// This module exists solely to provide a migration path for existing Docker
// products during the transition period.
package legacyadapters
