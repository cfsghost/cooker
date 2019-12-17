package module

type Event struct {
	Event string
	Payload interface{}
}

type Module struct {
	info *ModuleInfo
	iface interface{}
	moduleManager *ModuleManager
	externalModules map[string]interface{}
	eventChannel chan Event
}

func (m *Module) SetInterface(iface interface{}) {
	m.iface = iface
}

func (m *Module) GetInterface() interface{} {
	return m.iface
}

func (m *Module) SetupFuncAfterReady(fn func()) {
	m.moduleManager.SetupFuncAfterReady(m.info.Name, fn)
}

func (m *Module) SetupDependencies(dependencies []string) error {

	// Loading required modules
	modules, err := m.moduleManager.GetModules(dependencies)
	if err != nil {
		return err
	}

	m.externalModules = modules

	return nil
}

func (m *Module) GetExternalModule(name string) interface{} {

	value, err := m.moduleManager.GetModule(name)
	if err != nil {
		return nil
	}

	return value.GetInterface()
}

func (m *Module) GetEventChannel() (chan Event) {
	return m.eventChannel
}

func (m *Module) Emit(event Event) {

	go func() {
		m.eventChannel <- event
	}()
}
