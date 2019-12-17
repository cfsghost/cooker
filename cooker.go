package cooker

func CreateApp(orgName string, productName string, projectName string, programName string) *App {

	inst := new(App)

	inst.SetOrgName(orgName)
	inst.SetProductName(productName)
	inst.SetProjectName(projectName)
	inst.SetProgramName(programName)

	return inst
}
