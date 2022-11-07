/*
Copyright © 2022 Tyk Technologies
*/
package cmd

import (
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/TykTechnologies/gromit/mutex"
	"github.com/TykTechnologies/gromit/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var etcdPass, etcdHost, etcdUser, script string
var hasTLS bool
var caCertFile, clientCertFile, clientKeyFile string
var lock mutex.Lock

// mutexCmd represents the mutex command
var mutexCmd = &cobra.Command{
	Use:   "mutex",
	Short: "Interact with MaaS",
	Long: `The mutex as a service is backed by an external etcd cluster.
This command can be used to synchronise external processes.
`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var cli *clientv3.Client
		if hasTLS {
			ca, err := ioutil.ReadFile(caCertFile)
			if err != nil {
				log.Fatal().Err(err).Msg("couldn't read ca cert")
			}
			clientCert, err := ioutil.ReadFile(clientCertFile)
			if err != nil {
				log.Fatal().Err(err).Msg("couldn't read client cert")
			}
			clientKey, err := ioutil.ReadFile(clientKeyFile)
			if err != nil {
				log.Fatal().Err(err).Msg("couldn't read client key file")
			}
			tlsAuth := util.TLSAuthClient{
				CA:   ca,
				Cert: clientCert,
				Key:  clientKey,
			}
			tlsConfig, err := tlsAuth.GetTLSConfig()
			if err != nil {
				log.Fatal().Err(err).Msg("creating TLS config.")
			}
			cli, err = mutex.GetEtcdTLSClient(etcdHost, tlsConfig, 5)
			if err != nil {
				log.Fatal().Err(err).Msg("could not connect to etcd over TLS")
			}
		} else {
			var err error
			cli, err = mutex.GetEtcdClient(etcdHost, 5, etcdUser, etcdPass)
			if err != nil {
				log.Fatal().Err(err).Msg("could not connect to etcd")
			}
		}

		// create a new session
		sess, err := mutex.GetSession(cli)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to get a session lease")
		}

		// when session is closed lock on mutex will be released as well
		m := mutex.GetMutex(sess, mutex.ProdMutexPrefix+args[0])
		lock = mutex.Lock{
			Client:  cli,
			Session: sess,
			Mutex:   m,
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		err := lock.Close()
		if err != nil {
			log.Warn().Err(err).Msg("could not close session and/or client")
		}

	},
}

// getSubCmd represents the get subcommand from mutex command
var getSubCmd = &cobra.Command{
	Use:   "get <lock name>",
	Args:  cobra.MinimumNArgs(1),
	Short: "Acquire a lock named <lock name> and block until it is acquired.",
	Long:  `Implemented as a mutex in etcd named <lock name>. If it does not exist it will be created.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := lock.TryAcquire(); err == nil {
			// Simulate some processing
			op, err := exec.Command(script).CombinedOutput()
			if err != nil {
				lock.Release()
				log.Fatal().AnErr("error", err).Bytes("output", op).Msg("could not execute script")
			}
			log.Info().Bytes("output", op).Msg("script output")
			lock.Release()
		} else {
			log.Info().Msg("Environment being created by another process, exiting.")
			os.Exit(exitLockAlreadyTaken)
		}
	},
}

// initialization of variables
func init() {
	mutexCmd.PersistentFlags().BoolVar(&hasTLS, "tlsauth", false, "Use mTLS auth to connect to etcd, if this is set, --etcdpass and --etcduser are ignored")
	mutexCmd.PersistentFlags().StringVar(&caCertFile, "cacert", "", "The file containing the CA certificate")
	mutexCmd.PersistentFlags().StringVar(&clientCertFile, "cert", "", "The file containing the client certificate")
	mutexCmd.PersistentFlags().StringVar(&clientKeyFile, "key", "", "The file containing the client key")
	mutexCmd.MarkFlagsRequiredTogether("tlsauth", "cacert", "cert", "key")
	mutexCmd.PersistentFlags().StringVar(&etcdPass, "etcdpass", os.Getenv("ETCD_PASS"), "Password for etcd user")
	mutexCmd.PersistentFlags().StringVar(&etcdUser, "etcduser", "root", "etcd user to connect as")
	mutexCmd.MarkFlagsMutuallyExclusive("tlsauth", "etcduser")
	mutexCmd.PersistentFlags().StringVar(&etcdHost, "host", "", "etcd host")
	mutexCmd.PersistentFlags().StringVar(&script, "script", "testdata/mutex/script.sh", "script to be run after acquiring lock")

	mutexCmd.AddCommand(getSubCmd)
	rootCmd.AddCommand(mutexCmd)
}
