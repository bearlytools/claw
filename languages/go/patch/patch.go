// Package patch provides utilities for applying patches to Go source code.
// This allows you to create a patch that can be sent over the network or stored
// and then applied to modify a Claw Struct to update it. For large message that are mutated,
// this can be much more efficient than sending the entire modified message.
package patch
