package main

import (
	"os"
	"os/signal"
	"syscall"
)

func main() {

	a := initApplication()

	go a.listenForShutdown()

	err := a.App.ListenAndServe()
	a.App.ErrorLog.Println(err)
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

	a.App.InfoLog.Println("Application received signal", s.String())

	a.App.InfoLog.Println("Good bye!")

	os.Exit(0)
}
