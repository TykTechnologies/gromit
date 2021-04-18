package util

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"
)

type TLSAuthClient struct {
	CA   []byte
	Cert []byte
	Key  []byte
}

func (auth *TLSAuthClient) GetHTTPSClient() (http.Client, error) {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(auth.CA)

	cert, err := tls.X509KeyPair(auth.Cert, auth.Key)
	if err != nil {
		return http.Client{}, fmt.Errorf("(cert: [ %s ], key: [ %s ]): %w", string(auth.Cert), string(auth.Key), err)
	}
	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}
	if tlsConfig.NextProtos == nil {
		tlsConfig.NextProtos = []string{"http/1.1"}
	}
	tlsConfig.BuildNameToCertificate()

	return http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: time.Duration(10 * time.Second),
	}, nil
}

// String implements Stringer for printing
func (auth *TLSAuthClient) String() string {
	return fmt.Sprintf("(cert: [ %s ], key: [ %s ], ca: [ %s ])", string(auth.Cert), string(auth.Key), string(auth.CA))
}
