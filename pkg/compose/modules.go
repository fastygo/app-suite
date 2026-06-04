package compose

import (
	crmapp "github.com/fastygo/app-crm/pkg/app"
	gocmsapp "github.com/fastygo/app-gocms/pkg/app"
	modulemonitoring "github.com/fastygo/module-monitoring"
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
	modules := make([]contracts.Module, 0, 3)
	for _, bundle := range Bundles() {
		modules = append(modules, bundle.Module())
	}
	modules = append(modules, modulemonitoring.Module{})
	return modules
}
