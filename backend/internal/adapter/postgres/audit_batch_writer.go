// Package postgres holds the PostgreSQL adapter implementations of
// the repository ports. This file owns the batch audit writer wrapper
// — a write-side decorator that drains audit events into a buffered
// channel and flushes them to a wrapped `repository.AuditRepository`
// in groups, on a 5s tick OR a 100-event threshold (whichever first).
//
// Mission PERF-F3 (see /perf-audit.md).
//
// WIP STUB — implementation lands in the next commits.
package postgres

// reserved: BatchAuditWriter will be defined in audit_batch_writer.go.
