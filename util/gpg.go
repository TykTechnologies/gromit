package util

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
	"github.com/TykTechnologies/gromit/util/gpgagent"
	"strings"
	"encoding/hex"
	"fmt"
)


// GetSigningEntity gets the public keyring from gpg, using the agent
// Needs the gpgconf binary to find the agent socket path as gpg doesn't seem to set GPG_AUTH_INFO
// anymore. 
func GetSigningEntity(kid uint64) (*openpgp.Entity, error) {
	// Read keyring from gpg
	// signkeyReader, err := gpgBinary("--export-secret-keys", strKeyid)
	krFile := os.ExpandEnv("${HOME}/.gnupg/secring.gpg")
	signkeyReader, err := keyringFile(krFile)
	if err != nil {
		return nil, fmt.Errorf("cannot find keyring, gpg --export-secret-keys %#x > %s", kid, krFile)
	}
	entityList, err := openpgp.ReadKeyRing(signkeyReader)
	if err != nil {
		return nil, err
	}
	for i, e := range entityList {
		kid := e.PrimaryKey.KeyId
		log.Debug().Uint64("keyid", kid).Msgf("keyid %d (%#x) ", i, kid)
	}
	keys := entityList.KeysByIdUsage(kid, packet.KeyFlagSign)
	if len(keys) < 1 {
		return nil, fmt.Errorf("No signing key for keyid %d (%#x)", kid, kid)
	}
	key := entityList[0]
	if key.PrivateKey.Encrypted {
		passphrase, err := getPassphrase(hex.EncodeToString(key.PrivateKey.Fingerprint[:]))
		if err != nil {
			return nil, fmt.Errorf("getting passphrase from agent: %w", err)
		}
		err = key.PrivateKey.Decrypt(passphrase)
		if err != nil {
			return nil, err
		}
	}
	return key, nil
}

func getPassphrase(fp string) ([]byte, error) {
	agentSocket, err := gpgconfBinary("--list-dirs", "agent-socket")
	if err != nil {
		return nil, fmt.Errorf("getting agent socket: %w", err)
	}
	conn, err := gpgagent.NewGpgAgentConn(agentSocket)
	if err != nil {
		return nil, fmt.Errorf("connecting to gpg-agent: %w", err)
	}
	defer conn.Close()
	cacheID := strings.ToUpper(fp)
	// TODO: Add prompt, etc.
	request := gpgagent.PassphraseRequest{CacheKey: cacheID}
	passphrase, err := conn.GetPassphrase(&request)
	if err != nil {
		return nil, fmt.Errorf("getting passphrase: %w", err)
	}
	return []byte(passphrase), nil
}

func gpgconfBinary(args ...string) (string, error) {
	opBytes, err := exec.Command("gpgconf", args...).Output()
	if err != nil {
		return "", fmt.Errorf("cannot get output from gpgconf: %w", err)
	}
	return string(bytes.TrimRight(opBytes, "\n")), nil
}

func keyringFile(path string) (*os.File, error) {
	// Open the private key file
	krFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return krFile, nil
}
