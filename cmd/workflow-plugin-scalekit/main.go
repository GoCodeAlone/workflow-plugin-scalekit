package main

import (
	"github.com/GoCodeAlone/workflow-plugin-scalekit/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	sdk.Serve(internal.NewScalekitPlugin(), sdk.WithBuildVersion(sdk.ResolveBuildVersion(internal.Version)))
}
