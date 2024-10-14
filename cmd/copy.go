package cmd

import (
	"context"
	"fmt"
	"io"
	"regexp"

	"github.com/gobwas/glob"
	"github.com/pkg/errors"
)

var ErrReadCanceled = errors.WithMessage(io.ErrClosedPipe,
	"canceled before finishing read(s) from fifo")

type Pattern []Matcher

type Copier struct {
	rwc io.Reader
	ctx context.Context
	pat Pattern
}

type Matcher interface {
	Match([]byte) (bool, string)
}

type (
	Glob   struct{ glob.Glob }
	Regexp struct{ *regexp.Regexp }
)

func (g *Glob) Match(p []byte) (bool, string) {
	s := string(p)
	return g.Glob.Match(s), s
}

func (r *Regexp) Match(p []byte) (bool, string) {
	m := r.Regexp.Find(p)
	return m != nil, string(m)
}

func NewCopier(ctx context.Context, rwc io.Reader, pat ...string) (*Copier, error) {
	match := []Matcher{}
	for _, p := range pat {
		if len(p) >= 2 && p[0] == '/' && p[len(p)-1] == '/' {
			expr, err := regexp.Compile(p[1 : len(p)-1])
			if err != nil {
				return nil, errors.Wrapf(err, "invalid regular expression: %s", p)
			}
			match = append(match, &Regexp{expr})
		} else {
			expr, err := glob.Compile(p)
			if err != nil {
				return nil, errors.Wrapf(err, "invalid glob pattern: %s", p)
			}
			match = append(match, &Glob{expr})
		}
	}
	return &Copier{rwc, ctx, match}, nil
}

func (c *Copier) IsPatternDefined() bool {
	for _, p := range c.pat {
		if p != nil {
			return true
		}
	}
	return false
}

func (c *Copier) Match(p []byte) (bool, string) {
	for _, m := range c.pat {
		if ok, match := m.Match(p); ok {
			return true, match
		}
	}
	return false, ""
}

func (c *Copier) Read(p []byte) (n int, err error) {
	select {
	case <-c.ctx.Done():
		return 0, fmt.Errorf("%w: %w", ErrReadCanceled, c.ctx.Err())
	default:
		return c.rwc.Read(p)
	}
}
