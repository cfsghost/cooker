package cooker

import (
	"os"
	"os/signal"

	"fmt"
	"log"
	"github.com/spf13/viper"

	"cooker/module"
)

type AppInfo struct {
	orgName     string
	productName string
	projectName string
	programName string
}

type App struct {
	info AppInfo
	isRunning chan bool
	ModuleManager *module.ModuleManager
}

const organizationName = ""
const productName = "msghub"
const projectName = "room"
const programName = "room-message-receiver"

func CreateApp(orgName string, productName string, projectName string, programName string) {

	return &App{
		info: AppInfo{
			orgName: orgName,
			productName: productName,
			projectName: projectName,
			programName: programName,
		},
	}
}

func (app *App) Init() error {

	// Configuring config paths
	viper.SetConfigName(app.programName)
	viper.AddConfigPath("/etc/" + app.orgName + "/" + app.productName + "/" + app.projectName)
	viper.AddConfigPath("$HOME/.config/" + app.orgName + "/" + app.productName + "/" + app.projectName)
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s\n", err))
	}

	// Initializing state channel
	app.isRunning = make(chan bool)

	// Initializing module manager
	app.ModuleManager = &cooker.ModuleManager{}
	app.ModuleManager.Init(app)

	return nil
}

func (app *App) setInterruptHandler(sig int, fn func(*App)) {

	// Listen to system signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, sig)

	go func(){
		<-c

		fn(app)

		os.Exit(1)
	}()
}

func (app *App) GetModuleManager() *module.ModuleManager {
	return app.ModuleManager
}

func (app *App) Run() {

	// Run function of modules after ready
	funcs := app.GetModuleManager().GetFuncsAfterReady()
	for moduleName, fn := range funcs {
		log.Printf("Starting %s after ready", moduleName)
		go fn()
	}

	for {
		isRunning := <-app.isRunning
		if !isRunning {
			break
		}
	}
}
