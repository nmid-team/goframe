package nmid

import (
	"fmt"
	"goframe/pkg/util"
	"os"

	"github.com/HughNian/nmid/pkg/logger"

	wor "github.com/HughNian/nmid/pkg/worker"
)

type NmidWorker struct {
	Worker *wor.Worker
}

func InitWorker() *NmidWorker {
	nmidworker := &NmidWorker{}

	logger.Info("RunNmid Worker.")

	workerName := "goframe-worker"
	nmidServerAddr := fmt.Sprintf("%s:%s", os.Getenv("NMID_SERVER_HOST"), os.Getenv("NMID_SERVER_PORT"))
	logger.Info("|- worker name: %s", workerName)
	logger.Info("|- worker nmid server addr: %s", nmidServerAddr)

	var err error
	nmidworker.Worker = wor.NewWorker().SetWorkerName(workerName)
	err = nmidworker.Worker.AddServer("tcp", nmidServerAddr)
	if err != nil {
		logger.Error("new worker error %s", err.Error())
		nmidworker.Worker.WorkerClose()
		return nil
	}

	AddRouters(nmidworker.Worker, workerName)

	return nmidworker
}

func (nw *NmidWorker) RunWorker() {
	if err := nw.Worker.WorkerReady(); err != nil {
		logger.Error("worker ready error %s", err.Error())
		nw.Worker.WorkerClose()
		return
	}

	util.StartGo("nmid worker", func() {
		nw.Worker.WorkerDo()
	}, func(isdebug bool) {
		fmt.Println("start nmid worker over")
	})
}

func (nw *NmidWorker) CloseWorker() {
	nw.Worker.WorkerClose()
}
