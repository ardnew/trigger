package cmd

import (
	"bufio"
	"context"
	stderrors "errors"
	"io"
	"os"
	"sync"

	"github.com/ardnew/trigger/errs"
	"github.com/pkg/errors"
)

type Model struct {
	Cmd            string
	Args           []string
	Owrite, Append string
	Stdout         io.WriteCloser
	Stderr         io.WriteCloser
}

func (m *Model) SetCommandLine(line ...string) error {
	if len(line) == 0 {
		return errors.Wrapf(errs.ErrParseCmdLine, "unspecified command-line")
	}
	m.Cmd = line[0]
	if len(line) > 1 {
		m.Args = line[1:]
	}
	return nil
}

func (m *Model) output() (path string, flag int, defined bool) {
	if m.Owrite != "" {
		return m.Owrite, os.O_CREATE | os.O_WRONLY | os.O_TRUNC, true
	}
	return m.Append, os.O_CREATE | os.O_WRONLY | os.O_APPEND, m.Append != ""
}

func (m *Model) open(path string, flag int, fallback io.WriteCloser, selfDefined, otherDefined bool) error {
	if selfDefined {
		w, err := os.OpenFile(path, flag, 0o600)
		if err != nil {
			return err
		}
		m.Stdout = w
		m.Stderr = w
	} else {
		if otherDefined {
			m.Stdout = os.Stdout
			m.Stderr = os.Stderr
		} else {
			m.Stdout = fallback
			m.Stderr = fallback
		}
	}
	return nil
}

func (m *Model) Watch(
	ctx context.Context, in io.Reader, out io.WriteCloser, wait *sync.WaitGroup, notify chan<- string, pattern ...string,
) error {
	read, err := NewCopier(ctx, in, pattern...)
	if err != nil {
		return err
	}
	if read.IsPatternDefined() {
		scan := bufio.NewScanner(read)
		for err == nil {
			if !scan.Scan() {
				break
			}
			buf := scan.Bytes()
			if ok, match := read.Match(buf); ok {
				wait.Add(1)
				notify <- match
			}
			_, err = out.Write(buf)
		}
	} else {
		_, err = io.Copy(out, in)
	}
	return err
}

func OpenOutputs(mont, trig *Model) error {
	montOut, montFlag, montOK := mont.output()
	trigOut, trigFlag, trigOK := trig.output()
	if err := mont.open(montOut, montFlag, os.Stdout, montOK, trigOK); err != nil {
		return err
	}
	if err := trig.open(trigOut, trigFlag, os.Stderr, trigOK, montOK); err != nil {
		return err
	}
	return nil
}

func (m *Model) Close() (err error) {
	return stderrors.Join(m.Stdout.Close(), m.Stderr.Close())
}
