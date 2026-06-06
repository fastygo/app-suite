package compose

import (
	crmapp "github.com/fastygo/app-crm/pkg/app"
	gocmsapp "github.com/fastygo/app-gocms/pkg/app"
	modulechat "github.com/fastygo/module-chat"
	modulemonitoring "github.com/fastygo/module-monitoring"
	modulesupport "github.com/fastygo/module-support"
	"github.com/fastygo/platform/pkg/appbundle"
	"github.com/fastygo/platform/pkg/contracts"
)

func Bundles() []appbundle.Bundle {
	return []appbundle.Bundle{
		gocmsapp.Bundle(),
		crmapp.Bundle(),
	}
}

func DefaultModules() []contracts.Module {
	modules := make([]contracts.Module, 0, 5)
	for _, bundle := range Bundles() {
		modules = append(modules, bundle.Module())
	}
	modules = append(modules,
		modulemonitoring.Module{},
		modulesupport.Module{},
		modulechat.Module{},
	)
	return modules
}
