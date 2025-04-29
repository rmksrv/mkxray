package main

import (
	"strings"

	"github.com/muesli/termenv"
)

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

	headerLine := Header(app, app.Header)
	app.Lines = append(app.Lines, headerLine)

	for _, job := range app.Jobs {
		jobItem := UListItem(app, job.Name, job.Status, 0)
		app.Lines = append(app.Lines, jobItem)
		if job.Output != "" {
			jobOutputLines := Italics(app, strings.Split(job.Output, "\n"))
			app.Lines = append(app.Lines, jobOutputLines...)
		}
	}
}

func Header(app *App, s string) string {
	return app.Output.String(s).Bold().Foreground(termenv.ANSIWhite).String()
}

func UListItem(app *App, item string, status JobStatus, indent int) string {
	var marker string
	switch status {
	case OK:
		marker = app.Output.String("+").Foreground(termenv.ANSIGreen).String()
	case ERROR:
		marker = app.Output.String("x").Foreground(termenv.ANSIRed).String()
	case IN_PROGRESS:
		marker = app.Output.String("=").Foreground(termenv.ANSICyan).String()
	case WAITING:
		marker = app.Output.String("-").String()
	}
	indentation := strings.Repeat("  ", indent)
	return strings.Join([]string{indentation, marker, item}, " ")
}

func ErrorMsg(app *App, s string) string {
	return app.Output.String("ERROR:").Foreground(termenv.ANSIRed).String() + s
}

func Italics(app *App, s []string) []string {
	res := make([]string, len(s))
	for i, line := range s {
		res[i] = app.Output.String("      " + line).Bold().String()
	}
	return res
}
