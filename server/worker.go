package server

import (
	wor "github.com/HughNian/nmid/pkg/worker"
	"goframe/functions"
	"goframe/pkg/confer"
	"log"
)

const (
	DefaultName         = "DefaultWorker"
	DefaultNmidAddr     = "127.0.0.1:6808"
	DefaultSkyReportUrl = "192.168.64.6:30484"
)

// Worker nmid worker
var Worker *wor.Worker

func RunWorker() {
	println("RunNmid Worker.")

	workerName := confer.ConfigNmidGetString("workername", DefaultName)
	nmidServerAddr := confer.ConfigNmidGetString("serveraddr", DefaultNmidAddr)
	skyReporterUrl := confer.ConfigNmidGetString("skyreporterurl", DefaultSkyReportUrl)
	println("|- worker name:", workerName)
	println("|- worker nmid server addr:", nmidServerAddr)
	println("|- worker skyReport url:", skyReporterUrl)

	var err error
	Worker = wor.NewWorker().SetWorkerName(workerName).WithTrace(skyReporterUrl)
	//worker = wor.NewWorker().SetWorkerName(workerName)
	err = Worker.AddServer("tcp", nmidServerAddr)
	if err != nil {
		log.Fatalln(err)
		Worker.WorkerClose()
		return
	}

	functions.AddFunctions(Worker, workerName)

	if err = Worker.WorkerReady(); err != nil {
		log.Fatalln(err)
		Worker.WorkerClose()
		return
	}

	go Worker.WorkerDo()
}
