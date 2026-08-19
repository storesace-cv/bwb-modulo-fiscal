// Package fefixqueue persists FE fixture submissions and drives them through
// feboundary→femock using AGT test workbook identities (RM-FEFIX-007).
//
// States remain fixture_boundary_*; never authority_accepted or external_verified.
// Distinct from persistence/outbox→simulator (fiscaljws slice).
package fefixqueue
