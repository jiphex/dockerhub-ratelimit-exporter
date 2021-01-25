package rlex

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// RunMain is the main function, called from main()
func RunMain() {
	app := &cli.App{}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
