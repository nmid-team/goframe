package main

import (
	"goframe/nmid"
	"goframe/pkg/util"
	"goframe/script"
	"goframe/server"
	"os"

	"github.com/HughNian/nmid/pkg/logger"
	"github.com/joho/godotenv"

	"github.com/urfave/cli"
	_ "go.uber.org/automaxprocs"
)

func main() {
	app := cli.NewApp()
	app.Name = "goFrame"
	app.Usage = "run scripts!"
	app.Version = "0.0.1"
	app.Author = "hughnian"
	app.Commands = script.Commands()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "server",
			Value: "http&worker",
			Usage: "run server type:  http&worker",
		},
		cli.StringFlag{
			Name:  "c",
			Value: "/root/config/config.yaml",
			Usage: "config file url",
		},
	}
	godotenv.Load("./.env")

	nmidworker := nmid.InitWorker()
	nmidworker.RunWorker()
	app.Before = server.InitService
	app.Action = func(c *cli.Context) error {
		serverType := c.String("server")
		switch serverType {
		case "http":
			server.RunHTTP()
		case "worker":
			nmidworker.RunWorker()
		case "http&worker":
			{
				nmidworker.RunWorker()
				server.RunHTTP()
			}
		default:
			{
				nmidworker.RunWorker()
				server.RunHTTP()
			}
		}
		return nil
	}
	err := app.Run(os.Args)
	if err != nil {
		logger.Fatal("app run error:" + err.Error())
	}

	util.ListenAllGO(func() {
		nmidworker.CloseWorker()
	}, "goframe", "go over")
}
