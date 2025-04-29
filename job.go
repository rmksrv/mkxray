package main

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
