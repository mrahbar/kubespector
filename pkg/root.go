package pkg

import (
    "github.com/mrahbar/kubernetes-inspector/integration"
    "github.com/mrahbar/kubernetes-inspector/types"
)

var printer *integration.Printer
var cmdParams *types.CommandParams
var config types.Config

func initParams(cmdParams *types.CommandParams) {
    cmdParams = cmdParams
    printer = cmdParams.Printer
    config = cmdParams.Config
}
