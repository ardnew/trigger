package errs

import (
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

var ErrParseCmdLine = errors.New("parse command-line options")

func IsHelpFlag(err error) bool {
	return errors.Is(err, flag.ErrHelp)
}
