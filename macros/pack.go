package macros

import ()

// A pack is a collection of macros and settings.
type pack struct {
	Name   string
	Macros map[string]Macro
}
