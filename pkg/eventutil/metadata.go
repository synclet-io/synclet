// Package eventutil provides shared primitives used by every module's event
// envelope: metadata keys, encoder contracts, etc.
package eventutil

// MetadataKeyEventType is the key under which the concrete event-type string
// (e.g. "workspace.created") is stored on a txoutbox message's metadata map.
const MetadataKeyEventType = "event_type"
