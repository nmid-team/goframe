package script

import (
	"github.com/urfave/cli"
	"goframe/script/operator"
)

func Commands() []cli.Command {
	return []cli.Command{
		{
			Name:  "command_name",
			Usage: "当前用途",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name",
					Value: "value",
					Usage: "usage",
				},
			},
			Action: func(c *cli.Context) {
				operator.CatchCmdSignals()
				return
			},
		},
	}
}
