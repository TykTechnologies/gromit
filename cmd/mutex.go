/*
Copyright © 2022 Tyk Technologies
*/
package cmd

import (
	"os"
	"os/exec"

	"github.com/TykTechnologies/gromit/mutex"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var etcdPass, etcdHost, etcdUser, script string
var lock mutex.Lock

// mutexCmd represents the mutex command
var mutexCmd = &cobra.Command{
	Use:   "mutex",
	Short: "Interact with MaaS",
	Long: `The mutex as a service is backed by an external etcd cluster.
This command can be used to synchronise external processes.
`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// create client
		cli, err := mutex.GetEtcdClient(etcdHost, 5, etcdUser, etcdPass)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to etcd")
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
				log.Fatal().AnErr("error", err).Bytes("output", op).Msg("could not execute script")
			}
			log.Info().Bytes("output", op).Msg("script output")
			lock.Release()
		} else {
			log.Info().Msg("Environment being created by another process, exiting with no errors")
		}
	},
}

// initialization of variables
func init() {
	mutexCmd.PersistentFlags().StringVar(&etcdPass, "etcdpass", os.Getenv("ETCD_PASS"), "Password for etcd user")
	mutexCmd.PersistentFlags().StringVar(&etcdUser, "etcduser", "root", "etcd user to connect as")
	mutexCmd.PersistentFlags().StringVar(&etcdHost, "host", "ec2-3-66-86-193.eu-central-1.compute.amazonaws.com:2379", "etcd host")
	mutexCmd.PersistentFlags().StringVar(&script, "script", "testdata/mutex/script.sh", "script to be run after acquiring lock")

	mutexCmd.AddCommand(getSubCmd)
	rootCmd.AddCommand(mutexCmd)
}
