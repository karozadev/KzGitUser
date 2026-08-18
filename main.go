// Command kzgit helps developers visualize, verify and switch their active
// Git identity (user.name and user.email) to avoid committing under the
// wrong profile.
package main

import (
	"os"

	"github.com/karoza/kz-git-user/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
