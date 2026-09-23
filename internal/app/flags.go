package app

import (
	"flag"
	"strings"
)

// PositionalArg represents a non-flag CLI argument, recording whether it was passed after '--'.
type PositionalArg struct {
	Value   string
	Literal bool // True if passed after the '--' delimiter
}

// isHelpRequested returns true if args contains -h or --help before any '--' delimiter.
func isHelpRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

// parseFlagsAndPositional separates flags and positional arguments, respecting '--' and flag values.
// It parses the flag arguments into fs and returns the list of positional arguments.
func parseFlagsAndPositional(fs *flag.FlagSet, args []string) ([]PositionalArg, error) {
	var flagArgs []string
	var posArgs []PositionalArg
	inDoubleDash := false

	type boolFlag interface {
		IsBoolFlag() bool
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if inDoubleDash {
			posArgs = append(posArgs, PositionalArg{Value: arg, Literal: true})
			continue
		}
		if arg == "--" {
			inDoubleDash = true
			continue
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			// Check if it's in --flag=value format
			if strings.Contains(arg, "=") {
				flagArgs = append(flagArgs, arg)
				continue
			}

			// Check flag registration in fs
			flagName := strings.TrimLeft(arg, "-")
			f := fs.Lookup(flagName)
			if f != nil {
				if bf, ok := f.Value.(boolFlag); ok && bf.IsBoolFlag() {
					flagArgs = append(flagArgs, arg)
					continue
				}
				// Flag expects a value from the next token
				flagArgs = append(flagArgs, arg)
				if i+1 < len(args) {
					flagArgs = append(flagArgs, args[i+1])
					i++
				}
				continue
			}

			// Unknown flag: pass to flagArgs so fs.Parse can return the standard error
			flagArgs = append(flagArgs, arg)
			continue
		}

		// Positional argument
		posArgs = append(posArgs, PositionalArg{Value: arg, Literal: false})
	}

	if err := fs.Parse(flagArgs); err != nil {
		return nil, err
	}

	return posArgs, nil
}
