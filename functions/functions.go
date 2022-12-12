package functions

import (
	"errors"
	wor "github.com/HughNian/nmid/pkg/worker"
	"goframe/app/base/worker"
	"log"
)

func AddFunctions(w *wor.Worker, workerName string) {
	if nil == w {
		log.Fatalln(errors.New("worker not init"))
		return
	}

	w.AddFunction(workerName+`/`+worker.FuncNameDemo, worker.Demo)
}
