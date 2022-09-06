/*
Copyright © 2022 Tyk Technologies
*/
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/TykTechnologies/gromit/mutex"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var etcdPass, etcdHost, etcdUser string
var lock mutex.Lock

var mutexCmd = &cobra.Command{
	Use:   "mutex",
	Short: "Interact with MaaS",
	Long: `The mutex as a service is backed by an external etcd cluster.
This command can be used to synchronise external processes.
`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.Debug().Str("host", etcdHost).Str("user", etcdUser).Str("pass", etcdPass).Msg("etcd url")
		// create client
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   []string{etcdHost},
			DialTimeout: 5 * time.Second,
			Username:    etcdUser,
			Password:    etcdPass,
		})
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to etcd")
		}
		defer cli.Close()
		// create a new session
		sess, err := concurrency.NewSession(cli)
		if err != nil {
			fmt.Println(err)
		}
		// when session is closed lock on mutex will be released as well
		defer sess.Close()

		m := concurrency.NewMutex(sess, args[0])
		lock = mutex.Lock{
			Mutex: m,
		}
	},
}

var getSubCmd = &cobra.Command{
	Use:   "get <lock name>",
	Args:  cobra.MinimumNArgs(1),
	Short: "Acquire a lock named <lock name> and block until it is acquired.",
	Long:  `Implemented as a mutex in etcd named <lock name>. If it does not exist it will be created.`,
	Run: func(cmd *cobra.Command, args []string) {
		lock.Acquire()
		// Simulate some processing
		time.Sleep(15 * time.Second)
		lock.Release()
	},
}

func init() {
	mutexCmd.PersistentFlags().StringVar(&etcdPass, "etcdpass", os.Getenv("ETCD_PASS"), "Password for etcd user")
	mutexCmd.PersistentFlags().StringVar(&etcdUser, "etcduser", "root", "etcd user to connect as")
	mutexCmd.PersistentFlags().StringVar(&etcdHost, "host", "ec2-3-66-86-193.eu-central-1.compute.amazonaws.com:2379", "etcd host")

	mutexCmd.AddCommand(getSubCmd)
	rootCmd.AddCommand(mutexCmd)
}
