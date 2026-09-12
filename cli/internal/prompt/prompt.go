// Package prompt reads a short sequence of line and password inputs from
// the terminal.
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Reader prompts for input from os.Stdin, echoing prompts to out.
type Reader struct {
	in  *bufio.Reader
	out io.Writer
}

// NewReader wraps os.Stdin for a sequence of prompts. Create one per command
// invocation and reuse it for every field, so buffered input isn't dropped
// between prompts.
func NewReader(out io.Writer) *Reader {
	return &Reader{in: bufio.NewReader(os.Stdin), out: out}
}

// Line prompts for and returns one trimmed line of visible input.
func (r *Reader) Line(label string) (string, error) {
	fmt.Fprint(r.out, label)
	line, err := r.in.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// Hidden prompts for a line of input without echoing it, when stdin is a
// terminal. Otherwise (piped input, e.g. in scripts) it falls back to Line.
//
// On a real terminal this reads raw from the fd rather than through r.in's
// buffer, so input pasted ahead of the prompt (all fields at once) can be
// lost. Acceptable for a v1 skeleton; a Bubble Tea prompt replaces this later.
func (r *Reader) Hidden(label string) (string, error) {
	fmt.Fprint(r.out, label)
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return r.Line("")
	}
	b, err := term.ReadPassword(fd)
	fmt.Fprintln(r.out)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
