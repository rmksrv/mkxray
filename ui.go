package main

import (
	"os"
	"strings"

	"github.com/muesli/termenv"
)

var appInited = false

type App struct {
	Header         string
	Jobs           []Job
	Output         *termenv.Output
	RestoreConsole func() error
}

func InitApp(header string, jobs ...Job) *App {
	if appInited {
		panic("app already initialized")
	}

	restoreConsole, err := termenv.EnableVirtualTerminalProcessing(termenv.DefaultOutput())
	if err != nil {
		panic(err)
	}
	output := termenv.NewOutput(os.Stdout, termenv.WithProfile(termenv.TrueColor))

	appInited = true
	return &App{
		Header:         header,
		Jobs:           jobs,
		Output:         output,
		RestoreConsole: restoreConsole,
	}
}

func RenderUI(app *App, clear bool) {
	if clear {
		app.Output.ClearLines(len(app.Jobs) + 1)
	}

	println(header(app.Output, app.Header))

	for _, job := range app.Jobs {
		println(ulistItem(app.Output, job.Name, job.Status, 0))
	}
}

type Job struct {
	Name    string
	Status  JobStatus
	Execute func() error
}

func NewJob(name string, execute func() error) Job {
	return Job{
		Name:    name,
		Status:  WAITING,
		Execute: execute,
	}
}

func RunJob(job *Job) error {
	err := job.Execute()
	if err != nil {
		job.Status = ERROR
	} else {
		job.Status = OK
	}
	return err
}

type JobStatus int

const (
	OK JobStatus = iota
	ERROR
	WAITING
	IN_PROGRESS
)

func header(out *termenv.Output, s string) string {
	return out.String(s).Bold().Foreground(termenv.ANSIWhite).String()
}

func ulistItem(out *termenv.Output, item string, status JobStatus, indent int) string {
	var marker string
	switch status {
	case OK:
		marker = out.String("+").Foreground(termenv.ANSIGreen).String()
	case ERROR:
		marker = out.String("x").Foreground(termenv.ANSIRed).String()
	case IN_PROGRESS:
		marker = out.String("=").Foreground(termenv.ANSICyan).String()
	case WAITING:
		marker = out.String("-").String()
	}
	indentation := strings.Repeat("  ", indent)
	return strings.Join([]string{indentation, marker, item}, " ")
}

func ErrorMsg(app *App, s string) string {
	return app.Output.String("ERROR:").Foreground(termenv.ANSIRed).String() + s
}

func RenderEndMessage(app *App, ctx *XrayContext) {
	println()
	println(header(app.Output, "All jobs completed! Import the following link into your Xray client:"))
	println(ctx.VlessLink)
	println()
}
