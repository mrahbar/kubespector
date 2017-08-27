package pkg

import (
    "github.com/mrahbar/kubernetes-inspector/integration"
    "github.com/mrahbar/kubernetes-inspector/types"
)

var printer *integration.Printer
var cmdParams *types.CommandContext
var cmdExecutor types.CommandExecutor
var config types.Config

func initParams(commandContext *types.CommandContext) {
    commandContext = commandContext
    printer = commandContext.Printer
    config = commandContext.Config
    cmdExecutor = commandContext.CommandExecutor
}
