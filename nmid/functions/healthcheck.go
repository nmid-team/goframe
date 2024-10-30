package functions

import (
	"encoding/json"
	"fmt"

	"github.com/HughNian/nmid/pkg/model"
	wor "github.com/HughNian/nmid/pkg/worker"
	"github.com/vmihailenco/msgpack"
)

var (
	NameHealthCheck = "HealthCheck"
)

type Params struct {
	Health string `json:"health"`
}

type Res struct {
	State   int
	Message string
}

func HealthCheck(job wor.Job) (ret []byte, err error) {
	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	var params Params
	err = job.ShouldBind(&params)
	if err == nil {
		var resData []byte

		if params.Health == "check" {
			var res = Res{
				200,
				"success",
			}

			resData, _ = json.Marshal(&res)
		} else {
			var res = Res{
				403,
				"forbidden",
			}

			resData, _ = json.Marshal(&res)
		}

		retStruct := model.GetRetStruct()
		retStruct.Msg = "ok"
		retStruct.Data = resData
		ret, err := msgpack.Marshal(retStruct)
		if nil != err {
			return []byte(``), err
		}

		resp.RetLen = uint32(len(ret))
		resp.Ret = ret

		return ret, nil
	} else {
		return nil, fmt.Errorf("invalid params")
	}
}
