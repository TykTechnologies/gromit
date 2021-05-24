// +build !windows

package gpgagent

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

// NewGpgAgentConn connects to the GPG Agent as described in the
// GPG_AGENT_INFO environment variable.
func NewGpgAgentConn(sp string) (*Conn, error) {
	if len(sp) == 0 {
		return nil, ErrNoAgent
	}
	addr := &net.UnixAddr{Net: "unix", Name: sp}
	uc, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, err
	}
	br := bufio.NewReader(uc)
	lineb, err := br.ReadSlice('\n')
	if err != nil {
		return nil, err
	}
	line := string(lineb)
	if !strings.HasPrefix(line, "OK") {
		return nil, fmt.Errorf("gpgagent: didn't get OK; got %q", line)
	}
	return &Conn{uc, br}, nil
}
