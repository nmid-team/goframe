package util

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/HughNian/nmid/pkg/logger"
)

var (
	gonum     int64 = 0    //主协程数管理
	goState   int64 = 0    //游戏服运行状态 0运行中 1需要安全结束协程
	needAlarm       = true //需要发送告警信息
)

// 所有协程是否安全运行
func IsGoRuntime() bool {
	return atomic.LoadInt64(&goState) == 0
}

// go协程安全结束
func GoSecurityOver() {
	atomic.StoreInt64(&goState, 1)
}

// 开启一个主要协程 mark协程标识
func StartGo(mark string, f func(), overf func(isdebug bool)) {
	startGo(mark, true, f, overf)
}

// 次要go
// mark标识
// f 主要逻辑方法
// overf 结束方法
func StartMinorGO(mark string, f func(), overf func(isdebug bool)) {
	startGo(mark, false, f, overf)
}

// 开始go
// mark 标识
// ismain 是否主要协程(结束是否影响整体业务)
// f 主要方法
// overf 结束方法
func startGo(mark string, ismain bool, f func(), overf func(isdebug bool)) {
	if f == nil {
		log.Panicln("start server fail:" + mark + ", f is nil")
	}
	if ismain {
		atomic.AddInt64(&gonum, 1)
	}
	go func() {
		logger.Infof("start go: %s, ismain: %t", mark, ismain)
		if ismain {
			log.Println("start go:", mark)
		}
		defer func() {
			isdebug := false
			if err := recover(); err != nil {
				logger.Debug(fmt.Sprint("[debug] ", mark, " error:", err, " stack:", string(debug.Stack())))
				isdebug = true
			}
			if ismain {
				log.Println("end go:", mark, ",isdebug:", isdebug)
			}
			logger.Infof("server over mark: %v ,ismain: %t", mark, ismain)
			if overf != nil {
				func() { //防止结束任务debug
					defer func() {
						ListenDebug(mark + " overf bug")
					}()
					overf(isdebug)
				}()
			}
			if ismain { //需要安全结束协程
				atomic.AddInt64(&gonum, -1)
				GoSecurityOver()
			}
		}()
		f()
	}()
}

// 监听debug(true为有bug)
func ListenDebug(mark string) bool {
	if err := recover(); err != nil {
		logger.Debugf("[debug] %s  error: %s stack: %s", mark, err, string(debug.Stack()))
		return true
	}
	return false
}

// 监控所有主要协程，都结束后，才结束
// stopall 程序被kill后执行
func ListenAllGO(stopall func(), alarmGroup string, alarmContent string) {
	ListenKill()
	flag := false
	for {
		time.Sleep(2 * time.Second)
		if !flag && !IsGoRuntime() { //监听有协程关闭，主动停止需要手动停止的协程
			flag = true
			if stopall != nil {
				stopall()
			}
			log.Println("stopall goruntime")
		}
		v := atomic.LoadInt64(&gonum)
		if v <= 0 {
			if needAlarm {
				// 发送报警
				//SendAlarm(alarmGroup, alarmContent)
			}
			log.Println("all go over")
			return
		}
	}
}

// 监听kill命令
func ListenKill() {
	StartMinorGO("listen kill", func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		s := <-c
		needAlarm = false
		logger.Infof("Server Exit: %s", s.String())
		atomic.StoreInt64(&goState, 1) //提示监听协程结束
	}, nil)
}
