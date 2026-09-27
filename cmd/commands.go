package main

import (
	"fmt"
	"os"
)

type commandHandlerFunc func([]string)

var commandHandlers = map[string]commandHandlerFunc{
	"init":     handleInit,
	"capture":  handleCapture,
	"context":  handleContext,
	"sync":     handleSync,
	"resume":   handleResume,
	"repair":   handleRepair,
	"watch":    handleWatch,
	"search":   handleSearch,
	"artifact": handleArtifact,
	"resolve":  handleResolve,
	"history":  handleHistory,
	"timeline": handleTimeline,
	"diff":     handleDiff,
	"agent":    handleAgent,
	"export":   handleExport,
	"import":   handleImport,
	"handoff":  handleHandoff,
	"list":     handleList,
	"task":     handleTask,
	"project":  handleProject,
	"snapshot": handleSnapshot,
	"skill":    handleSkill,
	"config":   handleConfig,
}

func dispatchCommand(command string, args []string) {
	handler, ok := commandHandler(command)
	if !ok {
		fmt.Fprintln(os.Stderr, "Unknown command:", command)
		printUsage(false)
		os.Exit(1) // printUsage already prints instructions
	}
	handler(args)
}

func commandHandler(command string) (commandHandlerFunc, bool) {
	// Keep the lookup behind a function so the command registry is testable.
	handler, ok := commandHandlers[command]
	return handler, ok
}
