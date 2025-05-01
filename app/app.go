package main

import (
	"os"

	"github.com/muesli/termenv"
)

var appInited = false

type App struct {
	Header         string
	Jobs           []*Job
	Output         *termenv.Output
	RestoreConsole func() error
	Lines          []string
}

func InitApp(header string, jobs ...*Job) *App {
	if appInited {
		panic("app already initialized")
	}

	restoreConsole, err := termenv.EnableVirtualTerminalProcessing(termenv.DefaultOutput())
	if err != nil {
		panic(err)
	}
	output := termenv.NewOutput(os.Stdout, termenv.WithProfile(termenv.TrueColor))

	app := &App{
		Header:         header,
		Jobs:           jobs,
		Output:         output,
		RestoreConsole: restoreConsole,
		Lines:          make([]string, 0),
	}
	appInited = true
	return app
}
