package client

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// All paramters are filenames
func getMTLSClient(ca string, cert string, key string) (http.Client, error) {
	caCert, err := ioutil.ReadFile(ca)
	if err != nil {
		return http.Client{}, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cCert, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return http.Client{}, err
	}

	return http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cCert},
			},
		},
		Timeout: time.Duration(10 * time.Second),
	}, nil
}

func makeTLSRequest(client http.Client, url string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("could not construct request")
	}
	// if len(authToken) > 0 {
	// 	req.Header.Add("Authorization", authToken)
	// }
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal().Err(err).Msg("error in response")
	}
	defer resp.Body.Close()
	respContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Err(err).Msg("could not read response")
	}
	log.Info().Msg(string(respContent))
}
