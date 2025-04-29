package main

import (
	"os"
	"strings"

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

func ClearUI(app *App) {
	app.Output.ClearLines(len(app.Lines))
}

func RenderUI(app *App) {
	for _, line := range app.Lines {
		println(line)
	}
}

func RefreshLines(app *App) {
	app.Lines = make([]string, 0)

	headerLine := header(app.Output, app.Header)
	app.Lines = append(app.Lines, headerLine)

	for _, job := range app.Jobs {
		jobItem := ulistItem(app.Output, job.Name, job.Status, 0)
		app.Lines = append(app.Lines, jobItem)
		if job.Output != "" {
			jobOutputLines := italics(app.Output, strings.Split(job.Output, "\n"))
			app.Lines = append(app.Lines, jobOutputLines...)
		}
	}
}

type Job struct {
	Name    string
	Status  JobStatus
	Output  string
	Execute func() error
}

func NewJob(name string, execute func() error) *Job {
	return &Job{
		Name:    name,
		Status:  WAITING,
		Output:  "",
		Execute: execute,
	}
}

func RunJob(job *Job, app *App) error {
	job.Status = IN_PROGRESS
	ClearUI(app)
	RefreshLines(app)
	RenderUI(app)
	err := job.Execute()
	if err != nil {
		job.Status = ERROR
	} else {
		job.Status = OK
	}
	ClearUI(app)
	RefreshLines(app)
	RenderUI(app)
	return err
}

func ClearJobOutput(job *Job, app *App) {
	ClearUI(app)
	job.Output = ""
	RefreshLines(app)
	RenderUI(app)
}

func WriteJobOutput(output string, job *Job, app *App) {
	ClearUI(app)
	job.Output += output
	RefreshLines(app)
	RenderUI(app)
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

func AppendEndMessage(app *App, ctx *XrayContext) {
	app.Lines = append(app.Lines, "")
	app.Lines = append(app.Lines, header(app.Output, "All jobs completed! Import the following link into your Xray client:\n"))
	app.Lines = append(app.Lines, ctx.VlessLink)
	app.Lines = append(app.Lines, "")
}

func italics(output *termenv.Output, s []string) []string {
	res := make([]string, len(s))
	for i, line := range s {
		res[i] = output.String("      " + line).Bold().String()
	}
	return res
}
