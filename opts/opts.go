package opts

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ardnew/trigger/cmd"
	"github.com/ardnew/trigger/errs"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

var Output io.Writer = os.Stderr

type Model struct {
	*flag.FlagSet

	Monitor   cmd.Model
	Trigger   cmd.Model
	Retrigger bool

	PatternKey string
	Pattern    []string

	name, version string
	output        io.Writer
}

func New(name, version string, outputs ...io.Writer) *Model {
	output := Output
	if len(outputs) > 0 {
		output = io.MultiWriter(outputs...)
	}
	return &Model{
		FlagSet:    flag.NewFlagSet(name, flag.ContinueOnError),
		PatternKey: normalizeEnvKey(name) + "_PATTERN",
		name:       name,
		version:    version,
		output:     output,
	}
}

func (m *Model) Parse(args []string) error {
	m.StringVarP(&m.Monitor.Owrite, "monitor-output", "o", "", "write output from monitored command to `FILE`")
	m.StringVarP(&m.Monitor.Append, "monitor-append", "a", "", "append output from monitored command to `FILE`")
	m.BoolVarP(&m.Monitor.TeeWrites, "monitor-tee", "t", false, "tee all monitored command output to stdout")
	m.StringVarP(&m.Trigger.Owrite, "trigger-output", "O", "", "write output from triggered command to `FILE`")
	m.StringVarP(&m.Trigger.Append, "trigger-append", "A", "", "append output from triggered command to `FILE`")
	m.BoolVarP(&m.Trigger.TeeWrites, "trigger-tee", "T", false, "tee all triggered command output to stderr")
	m.BoolVarP(&m.Retrigger, "retrigger", "r", false, "rerun triggered command with each pattern found")
	m.SetInterspersed(true)
	out := os.Stdout
	m.SetOutput(out)
	m.Usage = func() {
		fmt.Fprintf(out, "%s version %s\n", m.name, m.version)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Usage:")
		fmt.Fprintf(out, "\t%s [OPTIONS] PATTERNS... -- MONITOR... ++ TRIGGER...\n", m.name)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Run a command and monitor its output for patterns. If found, then")
		fmt.Fprintln(out, "trigger another command. The captured pattern is then assigned to")
		fmt.Fprintf(out, "the environment variable `%s` for the triggered command.\n", m.PatternKey)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "The triggered command is run only once with the first matching pattern")
		fmt.Fprintln(out, "found in the monitored output (by default). Use the `--retrigger` flag")
		fmt.Fprintln(out, "to rerun the command with every occurrence of a matching pattern.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "The output of both commands can overwrite or append to files specified")
		fmt.Fprintln(out, "with command-line flags. The output streams are resolved as follows:")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  If no outputs are specified:")
		fmt.Fprintln(out, "   • Monitored command's stdout/stderr (combined)  →  inherited stdout")
		fmt.Fprintln(out, "   • Triggered command's stdout/stderr (combined)  →  inherited stderr")
		fmt.Fprintln(out, "  If only one command's output is specified:")
		fmt.Fprintln(out, "   • Specified command's stdout/stderr (combined)  →  specified output")
		fmt.Fprintln(out, "   • Opposite command's stdout                     →  inherited stdout")
		fmt.Fprintln(out, "   • Opposite command's stderr                     →  inherited stderr")
		fmt.Fprintln(out, "  If both commands' outputs are specified:")
		fmt.Fprintln(out, "   • Monitored command's stdout/stderr (combined)  →  specified output")
		fmt.Fprintln(out, "   • Triggered command's stdout/stderr (combined)  →  specified output")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "When a command's output is specified, it produces no output visible to")
		fmt.Fprintln(out, "the caller; everything is written to the specified output.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "However, each command can optionally tee their combined output stream so")
		fmt.Fprintln(out, "that its written to both the specified output and the caller's output")
		fmt.Fprintln(out, "simultaneously.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Similar to the case with no outputs specified, the destination stream")
		fmt.Fprintln(out, "depends on which command's output is being tee'd:")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "   • Monitored command's stdout/stderr (combined)  →  specified output")
		fmt.Fprintln(out, "      (with --monitor-tee)                          + inherited stdout")
		fmt.Fprintln(out, "   • Triggered command's stdout/stderr (combined)  →  specified output")
		fmt.Fprintln(out, "      (with --trigger-tee)                          + inherited stderr")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Options:")
		m.PrintDefaults()
	}
	patEnd, cmdEnd := -1, -1
	for i, arg := range args {
		switch arg {
		case "--":
			if patEnd < 0 {
				patEnd = i
			}
		case "++":
			if cmdEnd < 0 {
				cmdEnd = i
			}
		}
	}
	// The executable name has already been removed from args;
	// i.e., args[0] is the first argument after the executable name.
	var err error
	switch {
	case 0 > patEnd:
		err = errors.Wrapf(errs.ErrParseCmdLine, "end of PATTERNS delimiter not found: %q", "--")
	case 0 > cmdEnd:
		err = errors.Wrapf(errs.ErrParseCmdLine, "end of MONITOR command delimiter not found: %q", "++")
	case 0 >= patEnd: // args[:patEnd] may contain unparsed flags at this point
		err = errors.Wrap(errs.ErrParseCmdLine, "no PATTERNS specified")
	case patEnd+1 >= cmdEnd:
		err = errors.Wrap(errs.ErrParseCmdLine, "no MONITOR command specified")
	case cmdEnd+1 >= len(args):
		err = errors.Wrap(errs.ErrParseCmdLine, "no TRIGGER command specified")
	}
	if err != nil {
		if parseErr := m.FlagSet.Parse(args); parseErr != nil {
			return parseErr
		}
		return err
	}
	m.Monitor.Cmd, m.Monitor.Args = args[patEnd+1], args[patEnd+2:cmdEnd]
	m.Trigger.Cmd, m.Trigger.Args = args[cmdEnd+1], args[cmdEnd+2:]
	if err := m.FlagSet.Parse(args[:patEnd]); err != nil {
		return err
	}
	m.Pattern = m.Args()
	if len(m.Pattern) < 1 { // check again after all flags have been removed
		return errors.Wrap(errs.ErrParseCmdLine, "no PATTERNS specified")
	}
	return nil
}

func envKeyMap(legal string) func(r rune) rune {
	return func(r rune) rune {
		if strings.ContainsRune(legal, r) {
			return r
		}
		return '_'
	}
}

func normalizeEnvKey(s string) string {
	first := `ABCDEFGHIJKLMNOPQRSTUVWXYZ_`
	after := first + `0123456789`
	s = strings.ToUpper(s)
	if len(s) > 0 {
		s = string(envKeyMap(first)([]rune(s)[0])) +
			strings.Map(envKeyMap(after), s[1:])
	}
	return s
}
