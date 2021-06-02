// +build windows

package gpgagent

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os/exec"
	"strings"
)

// NewGpgAgentConn connects to the GPG Agent using the TCP socket on Windows that
// emulates the Unix socket.
func NewGpgAgentConn() (*Conn, error) {
	//In testing, this is the only reliable way I've found to get the path to the socket/nonce details
	agentOut, _ := exec.Command("gpg-connect-agent", "getinfo socket_name", "/bye").CombinedOutput()
	firstLine := strings.Split(string(agentOut), "\n")[0]
	if len(firstLine) < 3 {
		return nil, ErrNoAgent
	}

	//The output from gpg-connect-agent is a bit messy and needs cleaning up
	socketPath := strings.TrimSpace(firstLine[2:])
	sc, err := ioutil.ReadFile(socketPath)
	if err != nil {
		return nil, err
	}

	//The socket file is a port number, 0x0A, then a 16-bit nonce. This will extract
	//the port number and the nonce, and make sure that everything matches correctly.
	splitOn, _ := hex.DecodeString("0a")
	socketParts := bytes.SplitN(sc, splitOn, 2)
	if len(socketParts) != 2 {
		return nil, fmt.Errorf("Invalid socket file")
	}
	port := string(socketParts[0])
	nonce := socketParts[1]
	if len(nonce) != 16 {
		return nil, fmt.Errorf("Invalid socket file nonce")
	}

	//We have our socket number and nonce, so we can dial to localhost:port
	connAddr := fmt.Sprintf("127.0.0.1:%s", port)
	uc, err := net.Dial("tcp", connAddr)
	if err != nil {
		return nil, err
	}

	//Write the nonce to the socket and we should get our OK reply
	uc.Write(nonce)
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
