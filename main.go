package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/cidekar/adele-framework/httpserver"
	"github.com/cidekar/adele-framework/rpcserver"
)

var wg sync.WaitGroup

func main() {

	a := initApplication()

	go a.listenForShutdown()

	err := rpcserver.Start(a.App)
	if err != nil {
		log.Fatalf("failed to start rpc: %s", err)
	}

	a.jobsSchedule()

	//err = a.App.ListenAndServe()
	err = httpserver.Start(a.App)
	a.App.Log.Error(err)

}

// Here is where the wait group is invoked and all items in that were
// registered ask the application to wait until each task for the is done.
// These tasks will block the application until they are complete. For
// example, the application to wait until we have finished sending mail,
// add the mail to wg (i.e., wait group) and when complete call wg.Done()
func (a *application) listenForShutdown() {

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	s := <-quit

	a.App.Log.Info("Application received signal", s.String())

	err := rpcserver.Stop(a.App)
	if err != nil {
		log.Fatal("RPC server failed to stop:", err)
	}

	a.App.Log.Info("Good bye!")

	os.Exit(0)
}

// Here is where you may add jobs to the scheduler. Any jobs added will be
// called by the scheduler using the defined interval. You may use one of
// several pre-defined schedules in place of a cron expression (i.e., @yearly,
// @monthly, @weekly, @daily, @hourly and @every <duration>).
func (a *application) jobsSchedule() {
	// ...
}
