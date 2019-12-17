package cooker

import (
	"os"
	"os/signal"

	"fmt"
	"log"
	"github.com/spf13/viper"
	"github.com/cfsghost/cooker/module"
)

type Info struct {
	orgName     string
	productName string
	projectName string
	programName string
}

type App struct {
	info Info
	isRunning chan bool
	moduleManager *module.ModuleManager
}

func (app *App) SetOrgName(name string) {
	app.info.orgName = name
}

func (app *App) SetProductName(name string) {
	app.info.productName = name
}

func (app *App) SetProjectName(name string) {
	app.info.projectName = name
}

func (app *App) SetProgramName(name string) {
	app.info.programName = name
}

func (app *App) Init() error {

	// Configuring config paths
	viper.SetConfigName(app.info.programName)
	viper.AddConfigPath("/etc/" + app.info.orgName + "/" + app.info.productName + "/" + app.info.projectName)
	viper.AddConfigPath("$HOME/.config/" + app.info.orgName + "/" + app.info.productName + "/" + app.info.projectName)
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Warrning: failed to read confg file: %s\n", err)
	}

	// Initializing state channel
	app.isRunning = make(chan bool)

	// Initializing module manager
	moduleManager := module.CreateModuleManager()

	// Configuring module paths
	moduleManager.AddModulePath("./modules")
	moduleManager.AddModulePath("./out/modules")
	moduleManager.AddModulePath("/opt/" + app.info.orgName + "/" + app.info.productName + "/" + app.info.projectName + "/modules")

	app.moduleManager = moduleManager

	return nil
}

func (app *App) SetInterruptHandler(sig os.Signal, fn func(*App)) {

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
	return app.moduleManager
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
