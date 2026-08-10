// Package commands is Padinho's single command-composition root. Feature
// packages may expose registration functions and keep handlers together or in
// separate files as each feature warrants.
package commands

import "github.com/kiLeo13/SaladaFun/padinho/internal/command"

// Register declares every Padinho command on the unique registry.
// Padinho intentionally starts without commands.
func Register(_ *command.Registry) {}
