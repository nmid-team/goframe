package gin

import (
	"context"
	"errors"
	"github.com/HughNian/nmid/pkg/logger"
	"goframe/pkg/confer"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/DeanThompson/ginpprof"
	"github.com/gin-gonic/gin"
)

type OnShutdownF struct {
	f       func(cancel context.CancelFunc)
	timeout time.Duration
}

var (
	onShutdown []OnShutdownF
)

func RegisterOnShutdown(f func(cancel context.CancelFunc), timeout time.Duration) {
	onShutdown = append(onShutdown, OnShutdownF{
		f:       f,
		timeout: timeout,
	})
}

func NewGin() *gin.Engine {
	if confer.ConfigEnvIsDev() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(Recovery())
	if confer.ConfigEnvIsDev() {
		ginpprof.Wrap(r)
		r.Use(gin.Logger())
	}
	return r
}

func ListenHttp(httpPort string, r http.Handler, timeout int, f ...func()) {
	srv := &http.Server{
		Addr:    httpPort,
		Handler: r,
	}
	// 监听端口
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	// 注册关闭使用函数
	for _, v := range f {
		srv.RegisterOnShutdown(v)
	}
	// 监听信号
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGTERM)
	<-quit
	// 执行on shutdown 函数 - 同步
	for _, v := range onShutdown {
		var wg sync.WaitGroup
		wg.Add(1)
		ctx, cancel := context.WithCancel(context.TODO())
		go v.f(cancel)
		select {
		case <-time.After(v.timeout):
			log.Println("on shutdown timeout:", f)
			logger.Errorf("on shutdown timeout", errors.New("onShutdown fun err"))
			wg.Done()
		case <-ctx.Done():
			wg.Done()
		}
		wg.Wait()
	}
	// 执行shutdown
	log.Println("Shutdown Server ...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exiting")
}
