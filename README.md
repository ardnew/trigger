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

The triggered command is run only once with the first matching pattern
found in the monitored output (by default). Use the `--retrigger` flag
to rerun the command with every occurrence of a matching pattern.

The output of both commands can overwrite or append to files specified
with command-line flags. The output streams are resolved as follows:

  If no outputs are specified:
   • Monitored command's stdout/stderr (combined)  →  inherited stdout
   • Triggered command's stdout/stderr (combined)  →  inherited stderr
  If only one command's output is specified:
   • Specified command's stdout/stderr (combined)  →  specified output
   • Opposite command's stdout                     →  inherited stdout
   • Opposite command's stderr                     →  inherited stderr
  If both commands' outputs are specified:
   • Monitored command's stdout/stderr (combined)  →  specified output
   • Triggered command's stdout/stderr (combined)  →  specified output

When a command's output is specified, it produces no output visible to
the caller; everything is written to the specified output.

However, each command can optionally tee their combined output stream so
that its written to both the specified output and the caller's output
simultaneously.

Similar to the case with no outputs specified, the destination stream
depends on which command's output is being tee'd:

   • Monitored command's stdout/stderr (combined)  →  specified output
      (with --monitor-tee)                          + inherited stdout
   • Triggered command's stdout/stderr (combined)  →  specified output
      (with --trigger-tee)                          + inherited stderr

Options:
  -a, --monitor-append FILE   append output from monitored command to FILE
  -o, --monitor-output FILE   write output from monitored command to FILE
  -t, --monitor-tee           tee all monitored command output to stdout
  -r, --retrigger             rerun triggered command with each pattern found
  -A, --trigger-append FILE   append output from triggered command to FILE
  -O, --trigger-output FILE   write output from triggered command to FILE
  -T, --trigger-tee           tee all triggered command output to stderr
```
