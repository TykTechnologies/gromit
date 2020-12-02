/*
   Copyright © 2020 Tyk technologies

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
package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

// passwdCmd represents the passwd command
var passwdCmd = &cobra.Command{
	Use:   "passwd",
	Short: "Returns the password hash of the given plaintext",
	Long: `bcrypt.GenerateFromPassword([]byte(args[0]), 10)
does the business`,
	Run: func(cmd *cobra.Command, args []string) {
		newPassword, err := bcrypt.GenerateFromPassword([]byte(args[0]), 10)
		if err != nil {
			log.Err(err).Msg("password hash failed")
			os.Exit(1)
		}

		fmt.Println(string(newPassword))
	},
}

func init() {
	rootCmd.AddCommand(passwdCmd)
}
