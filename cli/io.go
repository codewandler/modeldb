package cli

import (
	"io"
	"os"
)

type IO struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

func (ioCfg IO) withDefaults() IO {
	if ioCfg.In == nil {
		ioCfg.In = os.Stdin
	}
	if ioCfg.Out == nil {
		ioCfg.Out = os.Stdout
	}
	if ioCfg.Err == nil {
		ioCfg.Err = os.Stderr
	}
	return ioCfg
}
