# trigger

## Usage

```
$ trigger -h
trigger version 0.1.0

Usage:
	trigger [OPTIONS] PATTERNS... -- MONITOR... ++ TRIGGER...

Run a command and monitor its output for patterns. If found, then
trigger another command. The captured pattern is then assigned to
the environment variable `TRIGGER_PATTERN` for the triggered command.

The triggered command is run only with the first matching pattern
found in the monitored output, by default. Use the `--retrigger`
flag to rerun the command with each pattern found.

The output of both commands can overwrite or append to files specified
with command-line flags. Both stdout and stderr are combined into a
single stream for each command. If no outputs are specified, the
combined stream of the monitored command is written to stdout, while
the triggered command's stream is written to stderr. If only one output
is specified, then the other command's stdout and stderr are attached
to the caller's stdout and stderr, respectively.

Options:
  -a, --monitor-append FILE   append output from monitored command to FILE
  -o, --monitor-output FILE   write output from monitored command to FILE
  -r, --retrigger             rerun triggered command with each pattern found
  -A, --trigger-append FILE   append output from triggered command to FILE
  -O, --trigger-output FILE   write output from triggered command to FILE
```
