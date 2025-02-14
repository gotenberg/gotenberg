package gotenberg

import (
	"runtime"
	"sort"
	"sync"

	flag "github.com/spf13/pflag"
)

// DebugInfo gathers data for debugging.
type DebugInfo struct {
	Version               string                            `json:"version"`
	Architecture          string                            `json:"architecture"`
	Modules               []string                          `json:"modules"`
	ModulesAdditionalData map[string]map[string]interface{} `json:"modules_additional_data"`
	Flags                 map[string]interface{}            `json:"flags"`
}

// BuildDebug builds the debug data from modules.
func BuildDebug(ctx *Context) {
	debugMu.Lock()
	defer debugMu.Unlock()

	debug = &DebugInfo{
		Version:               Version,
		Architecture:          runtime.GOARCH,
		Modules:               make([]string, len(ctx.moduleInstances)),
		ModulesAdditionalData: make(map[string]map[string]interface{}),
		Flags:                 make(map[string]interface{}),
	}

	i := 0
	for ID, mod := range ctx.moduleInstances {
		debug.Modules[i] = ID
		i++

		debuggable, ok := mod.(Debuggable)
		if !ok {
			continue
		}

		debug.ModulesAdditionalData[ID] = debuggable.Debug()
	}

	sort.Sort(AlphanumericSort(debug.Modules))

	ctx.ParsedFlags().VisitAll(func(f *flag.Flag) {
		debug.Flags[f.Name] = f.Value.String()
	})
}

// Debug returns the debug data.
func Debug() DebugInfo {
	debugMu.Lock()
	defer debugMu.Unlock()

	if debug == nil {
		return DebugInfo{}
	}

	return *debug
}

var (
	debug   *DebugInfo
	debugMu sync.Mutex
)
