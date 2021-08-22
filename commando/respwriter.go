package commando

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/scrapli/scrapligo/cfg"

	"github.com/fatih/color"
	"github.com/scrapli/scrapligo/driver/base"
)

const (
	filePermissions = 0755
)

type responseWriter interface {
	WriteResponse(r []interface{}, name string) error
}

func (app *appCfg) newResponseWriter(f string) responseWriter {
	switch f {
	case fileOutput:
		parentDir := "outputs"
		if app.timestamp {
			parentDir = parentDir + "_" + time.Now().Format(time.RFC3339)
		}

		app.outDir = parentDir

		return &fileWriter{
			parentDir,
		}
	case stdoutOutput:
		return &consoleWriter{}
	}

	return nil
}

// consoleWriter writes the scrapli responses to the console.
type consoleWriter struct{}

func (w *consoleWriter) writeFailure(name string) error {
	c := color.New(color.FgRed)
	c.Fprintf(
		os.Stderr,
		"\n**************************\n%s failed\n**************************\n",
		name,
	)

	return nil
}

func (w *consoleWriter) writeSuccess(r []interface{}, name string) error {
	c := color.New(color.FgGreen)
	c.Fprintf(os.Stderr, "\n**************************\n%s\n**************************\n", name)

	for _, mr := range r {
		switch respObj := mr.(type) {
		case *base.MultiResponse:
			for _, resp := range respObj.Responses {
				c := color.New(color.Bold)
				c.Fprintf(os.Stderr, "\n-- %s:\n", resp.ChannelInput)

				if resp.Failed {
					color.Set(color.FgRed)
				}

				fmt.Println(resp.Result)
			}
		case *cfg.Response:
			c := color.New(color.Bold)
			c.Fprintf(os.Stderr, "\n-- cfg-%s:\n", respObj.OperationType)

			if respObj.Failed {
				color.Set(color.FgRed)
			}

			fmt.Println(respObj.Result)
		case *cfg.DiffResponse:
			c := color.New(color.Bold)
			c.Fprint(os.Stderr, "\n-- cfg-DiffConfig:\n")

			if respObj.Failed {
				color.Set(color.FgRed)
			}

			fmt.Println(respObj.DeviceDiff)
		}
	}

	return nil
}

func (w *consoleWriter) WriteResponse(r []interface{}, name string) error {
	if r == nil {
		return w.writeFailure(name)
	}

	return w.writeSuccess(r, name)
}

// fileWriter writes the scrapli responses to the files on disk.
type fileWriter struct {
	dir string // output dir name
}

func (w *fileWriter) WriteResponse(r []interface{}, name string) error {
	outDir := path.Join(w.dir, name)
	if err := os.MkdirAll(outDir, filePermissions); err != nil {
		return err
	}

	for _, mr := range r {
		switch respObj := mr.(type) {
		case *base.MultiResponse:
			for _, resp := range respObj.Responses {
				c := sanitizeCmd(resp.ChannelInput)

				rb := []byte(resp.Result)
				if err := ioutil.WriteFile(path.Join(outDir, c), rb, filePermissions); err != nil {
					return err
				}
			}
		case *cfg.Response:
			fmt.Printf("GOT CFG RESPONSE, DUNNO HOW TO HANDLE YET!\n")
		case *cfg.DiffResponse:
			fmt.Printf("GOT DIFF RESPONSE, DUNNO HOW TO HANDLE YET!\n")
		}
	}

	return nil
}

func sanitizeCmd(s string) string {
	r := strings.NewReplacer(
		"/", "-",
		`\`, "-",
		`"`, ``,
		` `, `-`)

	return r.Replace(s)
}
