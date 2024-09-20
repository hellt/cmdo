package commando

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/scrapli/scrapligo/response"
	cfgresponse "github.com/scrapli/scrapligocfg/response"

	"github.com/fatih/color"
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
		case *response.MultiResponse:
			for _, resp := range respObj.Responses {
				c := color.New(color.Bold)
				c.Fprintf(os.Stderr, "\n-- %s:\n", resp.Input)

				if resp.Failed != nil {
					color.Set(color.FgRed)
				}

				fmt.Println(resp.Result)
			}
		case *cfgresponse.Response:
			c := color.New(color.Bold)
			c.Fprintf(os.Stderr, "\n-- cfg-%s:\n", respObj.Op)

			if respObj.Failed != nil {
				color.Set(color.FgRed)
			}

			fmt.Println(respObj.Result)
		case *cfgresponse.DiffResponse:
			c := color.New(color.Bold)
			c.Fprint(os.Stderr, "\n-- cfg-DiffConfig:\n")

			if respObj.Failed != nil {
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
		case *response.MultiResponse:
			for _, resp := range respObj.Responses {
				c := sanitizeFileName(resp.Input) // replace unsafe chars from a file name

				rb := []byte(resp.Result)
				if err := os.WriteFile(path.Join(outDir, c), rb, filePermissions); err != nil {
					return err
				}
			}
		case *cfgresponse.Response:
			rb := []byte(respObj.Result)
			if err := os.WriteFile(path.Join(outDir, respObj.Op), rb, filePermissions); err != nil {
				return err
			}
		case *cfgresponse.DiffResponse:
			rb := []byte(
				fmt.Sprintf("Device Diff:\n%s\n\nSide By Side Diff:\n%s\n\nUnified Diff:\n%s",
					respObj.DeviceDiff, respObj.SideBySideDiff(), respObj.UnifiedDiff()),
			)
			if err := os.WriteFile(path.Join(outDir, respObj.Op), rb, filePermissions); err != nil {
				return err
			}
		}
	}

	return nil
}

// sanitizeFileName ensures that file name doesn't contain invalid characters.
func sanitizeFileName(s string) string {
	// remove quotes and commas first
	r := strings.NewReplacer(
		`"`, ``,
		`'`, ``,
		`,`, ``)

	s = r.Replace(s)

	re := regexp.MustCompile(`[^0-9A-Za-z.\_\-]+`)

	return re.ReplaceAllString(s, "-")
}
