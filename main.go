package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var wg sync.WaitGroup

func main() {

	a := initApplication()

	go a.listenForShutdown()

	wg.Add(1)

	a.jobsSchedule()

	err := a.App.ListenAndServe()
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
