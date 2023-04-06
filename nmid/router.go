package nmid

import (
	"errors"
	"fmt"
	"log"

	"goframe/nmid/functions"

	wor "github.com/HughNian/nmid/pkg/worker"
)

func AddRouters(w *wor.Worker, workerName string) {
	if nil == w {
		log.Fatalln(errors.New("worker not init"))
		return
	}

	w.AddFunction(fmt.Sprintf("%s/%s", workerName, functions.NameDemo), functions.Demo)
}
