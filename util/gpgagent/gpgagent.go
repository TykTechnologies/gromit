/*
Copyright 2011 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package gpgagent interacts with the local GPG Agent.
package gpgagent

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"

	"io"
	"net/url"
	"os"
	"strings"
)

// Conn is a connection to the GPG agent.
type Conn struct {
	c  io.ReadWriteCloser
	br *bufio.Reader
}

var (
	ErrNoAgent = errors.New("GPG_AGENT_INFO not set in environment")
	ErrNoData  = errors.New("GPG_ERR_NO_DATA cache miss")
	ErrCancel  = errors.New("gpgagent: Cancel")
)

func (c *Conn) Close() error {
	c.br = nil
	return c.c.Close()
}

// PassphraseRequest is a request to get a passphrase from the GPG
// Agent.
type PassphraseRequest struct {
	CacheKey, Error, Prompt, Desc string

	// If the option --no-ask is used and the passphrase is not in
	// the cache the user will not be asked to enter a passphrase
	// but the error code GPG_ERR_NO_DATA is returned.  (ErrNoData)
	NoAsk bool
}

func (c *Conn) RemoveFromCache(cacheKey string) error {
	_, err := fmt.Fprintf(c.c, "CLEAR_PASSPHRASE %s\n", url.QueryEscape(cacheKey))
	if err != nil {
		return err
	}
	lineb, err := c.br.ReadSlice('\n')
	if err != nil {
		return err
	}
	line := string(lineb)
	if !strings.HasPrefix(line, "OK") {
		return fmt.Errorf("gpgagent: CLEAR_PASSPHRASE returned %q", line)
	}
	return nil
}

func (c *Conn) GetPassphrase(pr *PassphraseRequest) (passphrase string, outerr error) {
	defer func() {
		if e, ok := recover().(string); ok {
			passphrase = ""
			outerr = errors.New(e)
		}
	}()
	set := func(cmd string, val string) {
		if val == "" {
			return
		}
		_, err := fmt.Fprintf(c.c, "%s %s\n", cmd, val)
		if err != nil {
			panic("gpgagent: failed to send " + cmd)
		}
		line, _, err := c.br.ReadLine()
		if err != nil {
			panic("gpgagent: failed to read " + cmd)
		}
		if !strings.HasPrefix(string(line), "OK") {
			panic("gpgagent: response to " + cmd + " was " + string(line))
		}
	}
	if d := os.Getenv("DISPLAY"); d != "" {
		set("OPTION", "display="+d)
	}
	tty, err := os.Readlink("/proc/self/fd/0")
	if err == nil {
		set("OPTION", "ttyname="+tty)
	}

	ttyType := os.Getenv("TERM")
	if len(ttyType) == 0 {
		ttyType = "vt100"
	}
	set("OPTION", "ttytype="+ttyType)

	opts := ""
	if pr.NoAsk {
		opts += "--no-ask "
	}

	encOrX := func(s string) string {
		if s == "" {
			return "X"
		}
		return url.QueryEscape(s)
	}

	_, err = fmt.Fprintf(c.c, "GET_PASSPHRASE %s%s %s %s %s\n",
		opts,
		url.QueryEscape(pr.CacheKey),
		encOrX(pr.Error),
		encOrX(pr.Prompt),
		encOrX(pr.Desc))
	if err != nil {
		return "", err
	}
	lineb, err := c.br.ReadSlice('\n')
	if err != nil {
		return "", err
	}
	line := string(lineb)
	if strings.HasPrefix(line, "OK ") {
		decb, err := hex.DecodeString(line[3 : len(line)-1])
		if err != nil {
			return "", err
		}
		return string(decb), nil
	}
	fields := strings.Split(line, " ")
	if len(fields) >= 2 && fields[0] == "ERR" {
		switch fields[1] {
		case "67108922":
			return "", ErrNoData
		case "83886179":
			return "", ErrCancel
		}
	}
	return "", errors.New(line)
}
