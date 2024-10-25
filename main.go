package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/ardnew/trigger/cmd"
	"github.com/ardnew/trigger/errs"
	"github.com/ardnew/trigger/opts"
)

const version = "0.3.0"

const badExe = "%!s(BADEXE)"

func exeName() string {
	e, err := os.Executable()
	if err != nil {
		return badExe
	}
	return filepath.Base(e)
}

type count struct {
	*sync.Mutex
	n int
}

func (c *count) inc(n int) int {
	c.Lock()
	defer c.Unlock()
	c.n += n
	return c.n
}

func (c *count) get() int {
	c.Lock()
	defer c.Unlock()
	return c.n
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opt := opts.New(exeName(), version)
	num := count{ Mutex: new(sync.Mutex) }

	if err := opt.Parse(os.Args[1:]); err != nil {
		if errs.IsHelpFlag(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	if err := cmd.OpenOutputs(&opt.Monitor, &opt.Trigger); err != nil {
		fmt.Fprintf(os.Stderr, "%v: %s", err, "open monitor and trigger outputs")
		os.Exit(2)
	}

	mont := exec.CommandContext(ctx, opt.Monitor.Cmd, opt.Monitor.Args...)
	// mont.Stdout = opt.Monitor.Stdout
	// mont.Stderr = opt.Monitor.Stderr

	stdout, err := mont.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v: %s", err, "open monitor stdout pipe")
		os.Exit(5)
	}
	stderr, err := mont.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v: %s", err, "open monitor stderr pipe")
		os.Exit(6)
	}

	var wg sync.WaitGroup
	notify := make(chan string)

	go func(cn *count, env []string) {
		trigger := true
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-notify:
				cn.inc(1)
				if trigger {
					trigger = opt.Retrigger
					trig := exec.CommandContext(ctx, opt.Trigger.Cmd, opt.Trigger.Args...)
					trig.Stdout = opt.Trigger.Stdout
					trig.Stderr = opt.Trigger.Stderr
					trig.Env = append(env, fmt.Sprintf("%s=%s", opt.PatternKey, msg))
					if err := trig.Run(); err != nil {
						fmt.Fprintf(os.Stderr, "%v: %s", err, "run trigger command")
						os.Exit(7)
					}
				}
				wg.Done()
			}
		}
	}(&num, os.Environ())

	if opt.Monitor.Stdout == opt.Monitor.Stderr {
		wg.Add(1)
		go func(group *sync.WaitGroup) {
			err := opt.Monitor.Watch(ctx, io.MultiReader(stdout, stderr), opt.Monitor.Stdout, &wg, notify, opt.Pattern...)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v: %s", err, "watch monitor command")
				os.Exit(8)
			}
			wg.Done()
		}(&wg)
	} else {
		wg.Add(2)
		go func(group *sync.WaitGroup) {
			err := opt.Monitor.Watch(ctx, stdout, opt.Monitor.Stdout, &wg, notify, opt.Pattern...)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v: %s", err, "watch monitor command")
				os.Exit(8)
			}
			wg.Done()
		}(&wg)

		go func(group *sync.WaitGroup) {
			err := opt.Monitor.Watch(ctx, stderr, opt.Monitor.Stderr, &wg, notify, opt.Pattern...)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v: %s", err, "watch monitor command")
				os.Exit(8)
			}
			wg.Done()
		}(&wg)
	}

	if err := mont.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%v: %s", err, "start monitor command")
		os.Exit(9)
	}

	wg.Wait()

	if err := mont.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "%v: %s", err, "terminate monitor command")
		os.Exit(10)
	}

	if num.get() == 0 {
		os.Exit(0x7F)
	}
}
