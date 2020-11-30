package util

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type TLSAuthClient struct {
	CA   string
	Cert string
	Key  string
}

func (auth *TLSAuthClient) GetHTTPClient() (http.Client, error) {
	caCert, err := ioutil.ReadFile(auth.CA)
	if err != nil {
		return http.Client{}, fmt.Errorf("%s: %w", auth.CA, err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(auth.Cert, auth.Key)
	if err != nil {
		return http.Client{}, fmt.Errorf("(%s, %s): %w", auth.Cert, auth.Key, err)
	}

	return http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cert},
			},
		},
		Timeout: time.Duration(10 * time.Second),
	}, nil
}
