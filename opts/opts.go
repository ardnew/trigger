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
	m.StringVarP(&m.Trigger.Owrite, "trigger-output", "O", "", "write output from triggered command to `FILE`")
	m.StringVarP(&m.Trigger.Append, "trigger-append", "A", "", "append output from triggered command to `FILE`")
	m.BoolVarP(&m.Retrigger, "retrigger", "r", false, "rerun triggered command with each pattern found")
	m.SetInterspersed(true)
	m.SetOutput(m.output)
	m.Usage = func() {
		fmt.Fprintf(m.output, "%s version %s\n", m.name, m.version)
		fmt.Fprintln(m.output)
		fmt.Fprintln(m.output, "Usage:")
		fmt.Fprintf(m.output, "\t%s [OPTIONS] PATTERNS... -- MONITOR... ++ TRIGGER...\n", m.name)
		fmt.Fprintln(m.output)
		fmt.Fprintln(m.output, "Run a command and monitor its output for patterns. If found, then")
		fmt.Fprintln(m.output, "trigger another command. The captured pattern is then assigned to")
		fmt.Fprintf(m.output, "the environment variable `%s` for the triggered command.\n", m.PatternKey)
		fmt.Fprintln(m.output)
		fmt.Fprintln(m.output, "The triggered command is run only with the first matching pattern")
		fmt.Fprintln(m.output, "found in the monitored output, by default. Use the `--retrigger`")
		fmt.Fprintln(m.output, "flag to rerun the command with each pattern found.")
		fmt.Fprintln(m.output)
		fmt.Fprintln(m.output, "The output of both commands can overwrite or append to files specified")
		fmt.Fprintln(m.output, "with command-line flags. Both stdout and stderr are combined into a")
		fmt.Fprintln(m.output, "single stream for each command. If no outputs are specified, the")
		fmt.Fprintln(m.output, "combined stream of the monitored command is written to stdout, while")
		fmt.Fprintln(m.output, "the triggered command's stream is written to stderr. If only one output")
		fmt.Fprintln(m.output, "is specified, then the other command's stdout and stderr are attached")
		fmt.Fprintln(m.output, "to the caller's stdout and stderr, respectively.")
		fmt.Fprintln(m.output)
		fmt.Fprintln(m.output, "Options:")
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
