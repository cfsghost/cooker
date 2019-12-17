package module

import (
	"fmt"
	"log"
	"reflect"
	"errors"
	"io/ioutil"
	"path"
	"path/filepath"
	"plugin"
	"github.com/spf13/viper"
	"logagent/sdk/module"
)

type ModuleManager struct {
	app *App
	modulePaths []string
	modules map[string]*ModuleInfo
	afterReady map[string]func()
}

type ModuleInfo struct {
	Name string
	FilePath string
	Plugin *plugin.Plugin
	Instance *Module
}

func (mg *ModuleManager) Init(app *App) {

	mg.app = app;
	mg.modules = make(map[string]*ModuleInfo)
	mg.afterReady = make(map[string]func())

	// Module paths
	mg.AddModulePath("./modules")
	mg.AddModulePath("./out/modules")
	mg.AddModulePath("/opt/" + organizationName + "/" + productName + "/" + projectName + "/modules")

	// Load basic modules
	modules := viper.GetStringSlice("general.modules")
	mg.LoadModules(modules)
}

func (mg *ModuleManager) AddModulePath(modulePath string) {
	mg.modulePaths = append(mg.modulePaths, modulePath)
}

func (mg *ModuleManager) LoadModules(names []string) {

	for _, moduleName := range names {

		// Check whether module loaded already or not
		if mg.Exists(moduleName) {
			continue
		}

		// Search module file
		moduleInfo, err := mg.SearchModule(moduleName)
		if err != nil {
			log.Println(err)
			continue
		}

		// Register module
		mg.Register(moduleInfo.Name, moduleInfo)
	}
}

func (mg *ModuleManager) Exists(moduleName string) bool {

	_, ok := mg.modules[moduleName]
	if !ok {
		return false
	}

	return true
}

func (mg *ModuleManager) SearchModule(moduleName string) (*ModuleInfo, error) {

	log.Printf("Searching module: %s", moduleName);

	for _, modulePath := range mg.modulePaths {
		moduleInfo, err := mg.FindModule(moduleName, modulePath)
		if err == nil {
			return moduleInfo, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("Cannot find module: %s", moduleName))
}

func (mg *ModuleManager) FindModule(moduleName string, modulePath string) (*ModuleInfo, error) {

	log.Printf("Scanning %s", modulePath);

	// Searching modules
	files, err := ioutil.ReadDir(modulePath)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	for _, file := range files {

		// Check extension
		if filepath.Ext(file.Name()) != ".so" {
			continue
		}

		// Load module file
		fullpath := path.Join(modulePath, file.Name())

		//log.Printf("Opening %s ...", fullpath)
		moduleInfo, err := mg.Load(fullpath, moduleName)
		if err != nil {
			log.Println(err)
			continue
		}

		// Doesn't match
		if moduleInfo == nil {
			continue
		}

		// Initializing module
		moduleInfo, err = mg.InitModule(moduleInfo)
		if err != nil {
			log.Println(err)
			log.Printf("Failed to load module: %s", fullpath)
			continue
		}

		log.Printf("Found module: %s", moduleName)

		return moduleInfo, nil
	}

	return nil, errors.New("Cannot find module")
}

func (mg *ModuleManager) InitModule(moduleInfo *ModuleInfo) (*ModuleInfo, error) {

	// Lookup symbol
	symbol, err := moduleInfo.Plugin.Lookup("InitModule")
	if err != nil {
		return nil, err
	}

	// Create a module prototype
	module := &Module{
		info: moduleInfo,
		moduleManager: mg,
		eventChannel: make(chan sdk_module.Event),
	}

	// Getting initializer
	initializer := symbol.(func(sdk_module.ModuleCore) (interface{}, error))
	moduleIface, err := initializer(module)
	if err != nil {
		return nil, err
	}

	module.SetInterface(moduleIface)

	moduleInfo.Instance = module

	return moduleInfo, nil
}

func (mg *ModuleManager) Load(modulePath string, moduleName string) (*ModuleInfo, error) {

	// Open
	p, err := plugin.Open(modulePath)
	if err != nil {
		return nil, err
	}

	// Lookup symbol to check module name
	symName, err := p.Lookup("ModuleName")
	if err != nil {
		return nil, err
	}

	// Getting module name
	var name *string
	name, ok := symName.(*string)
	if !ok {
		return nil, errors.New("Failed to get module name")
	}

	if *name != moduleName {
		return nil, nil
	}

	return &ModuleInfo{
		Name: moduleName,
		FilePath: modulePath,
		Plugin: p,
	}, nil
}

func (mg *ModuleManager) GetModule(name string) (*Module, error) {

	moduleInfo, ok := mg.modules[name]
	if !ok {
		return nil, errors.New("No such module")
	}

	return moduleInfo.Instance, nil
}

func (mg *ModuleManager) GetModules(names []string) (map[string]interface{}, error) {

	modules := make(map[string]interface{}, len(names))

	for _, moduleName := range names {

		// Check whether module loaded already or not
		if !mg.Exists(moduleName) {
			return nil, errors.New(fmt.Sprintf("No such module: %s", moduleName))
		}

		value, err := mg.GetModule(moduleName)
		if err != nil {
			return nil, err
		}

		modules[moduleName] = value.GetInterface()
	}

	return modules, nil
}

func (mg *ModuleManager) Register(name string, moduleInfo *ModuleInfo) {

	module := reflect.ValueOf(moduleInfo.Instance.GetInterface())

	// Initializing module
	initialize := module.MethodByName("Initialize")
	if !initialize.IsValid() {
		log.Printf("Cannot register module: %s", name)
		return
	}

	returnValues := initialize.Call([]reflect.Value{})
	if returnValues[0].Interface() != nil {
		log.Println(returnValues[0].Interface())
		log.Printf("Cannot register module: %s", name)
		return
	}

	mg.modules[name] = moduleInfo
}

func (mg *ModuleManager) Unregister(name string) {

	value, err := mg.GetModule(name)
	if err != nil {
		log.Printf("No such module \"%s\"", name)
		return
	}

	module := reflect.ValueOf(value)

	// Uninitializing
	uninitialize := module.MethodByName("Uninitialize")
	if uninitialize.IsValid() {
		uninitialize.Call([]reflect.Value{})
	}
}

func (mg *ModuleManager) Broadcast(eventName string, payload interface{}) {

	// TODO
/*
	event := sdk_module.Event{
		Event: eventName,
		Payload: payload,
	}

	for moduleName, moduleInfo := range mg.modules {
		log.Printf("Dispatching event to module: %s", moduleName)
		moduleInfo.Instance.Emit(event)
	}
*/
}

func (mg *ModuleManager) SetupFuncAfterReady(moduleName string, fn func()) {
	mg.afterReady[moduleName] = fn
}

func (mg *ModuleManager) GetFuncsAfterReady() map[string]func() {
	return mg.afterReady
}
