package commando

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/scrapli/scrapligo/driver/base"
)

type responseWriter interface {
	WriteResponse(r *base.MultiResponse, name string, d *device, appCfg *appCfg) error
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

func (w *consoleWriter) WriteResponse(r *base.MultiResponse, name string, d *device, appCfg *appCfg) error {
	c := color.New(color.FgGreen)
	c.Fprintf(os.Stderr, "\n**************************\n%s\n**************************\n", name)

	for idx, cmd := range d.SendCommands {
		c := color.New(color.Bold)
		c.Fprintf(os.Stderr, "\n-- %s:\n", cmd)

		if r.Responses[idx].Failed {
			color.Set(color.FgRed)
		}

		fmt.Println(r.Responses[idx].Result)
	}

	return nil
}

// fileWriter writes the scrapli responses to the files on disk.
type fileWriter struct {
	dir string // output dir name
}

func (w *fileWriter) WriteResponse(r *base.MultiResponse, name string, d *device, appCfg *appCfg) error {
	outDir := path.Join(w.dir, name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	for idx, cmd := range d.SendCommands {
		c := sanitizeCmd(cmd)

		rb := []byte(r.Responses[idx].Result)
		if err := ioutil.WriteFile(path.Join(outDir, c), rb, 0755); err != nil { //nolint:gosec
			return err
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
