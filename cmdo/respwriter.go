package cmdo

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/fatih/color"
	"github.com/scrapli/scrapligo/driver/base"
)

type responseWriter interface {
	WriteResponse(r *base.MultiResponse, name string, d device) error
}

func newResponseWriter(f string) (responseWriter, error) {
	switch f {
	case "file":
		return &fileWriter{}, nil
	case "stdout":
		return &consoleWriter{}, nil
	}
	return nil, nil
}

// consoleWriter writes the scrapli responses to the console
type consoleWriter struct{}

func (w *consoleWriter) WriteResponse(r *base.MultiResponse, name string, d device) error {
	color.Green("\n**************************\n%s\n**************************\n", name)
	for idx, cmd := range d.SendCommands {
		c := color.New(color.Bold)
		c.Printf("\n-- %s:\n", cmd)
		fmt.Println(r.Responses[idx].Result)
	}
	return nil
}

// fileWriter writes the scrapli responses to the files on disk
type fileWriter struct{}

func (w *fileWriter) WriteResponse(r *base.MultiResponse, name string, d device) error {
	outDir := path.Join("outputs", name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	for idx, cmd := range d.SendCommands {
		c := sanitizeCmd(cmd)
		rb := []byte(r.Responses[idx].Result)
		if err := ioutil.WriteFile(path.Join(outDir, c), rb, 0755); err != nil {
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
