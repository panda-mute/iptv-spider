package router

import "github.com/kataras/iris/v12"

func registerMacros(app *iris.Application) {
	// has
	//app.Macros().Get("string").RegisterFunc("has", func(validNames []string) func(string) bool {
	//	return func(paramValue string) bool {
	//		for _, validName := range validNames {
	//			if validName == paramValue {
	//				return true
	//			}
	//		}
	//		return false
	//	}
	//})
}
