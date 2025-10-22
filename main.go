package main

import (
	"log"
	"myapp/handlers"
	"myapp/middleware"
	"myapp/models"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/cidekar/adele-framework"
	"github.com/cidekar/adele-framework/httpserver"
	"github.com/cidekar/adele-framework/provider"
	"github.com/cidekar/adele-framework/rpcserver"
)

var wg sync.WaitGroup

func main() {

	a := bootstrapApplication()

	go a.Mail.ListenForMail()

	go a.listenForShutdown()

	err := rpcserver.Start(a.App)
	if err != nil {
		log.Fatalf("failed to start rpc: %s", err)
	}

	a.jobsSchedule()

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

func bootstrapApplication() *application {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	a := &adele.Adele{}
	err = a.New(path)
	if err != nil {
		log.Fatal(err)
	}

	a.AppName = "myapp"

	myMiddleware := &middleware.Middleware{
		App: a,
	}

	myHandlers := &handlers.Handlers{
		App: a,
	}

	app := &application{
		App:        a,
		Handlers:   myHandlers,
		Mail:       &a.Mail,
		Middleware: myMiddleware,
	}

	app.App.Routes = app.routes()

	app.Models = models.New(a)

	p := &provider.Provider{
		EnabledProviders: make(map[string]bool),
		ProviderConfigs:  make(map[string]map[string]interface{}),
	}

	a.Provider = p

	if err := a.Provider.LoadProviders(app.App); err != nil {
		a.Log.Error(err)
		os.Exit(1)
	}

	return app
}
