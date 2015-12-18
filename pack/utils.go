package pack
import (
	"os"
	"io/ioutil"
	"log"
	"io"
	"bytes"
"text/template"
)
// Copies file source to destination dest.
func CopyFile(source string, dest string) (err error) {
	sf, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sf.Close()
	df, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer df.Close()
	_, err = io.Copy(df, sf)
	if err == nil {
		si, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, si.Mode())
		}

	}

	return
}

// Recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory must *not* exist.
func CopyDir(source string, dest string) (err error) {

	// get properties of source dir
	fi, err := os.Stat(source)
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		return &CustomError{"Source is not a directory"}
	}

	// create dest dir

	err = os.MkdirAll(dest, fi.Mode())
	if err != nil {
		return err
	}

	entries, err := ioutil.ReadDir(source)

	for _, entry := range entries {

		sfp := source + "/" + entry.Name()
		dfp := dest + "/" + entry.Name()
		if entry.IsDir() {
			err = CopyDir(sfp, dfp)
			if err != nil {
				log.Println(err)
			}
		} else {
			// perform copy
			err = CopyFile(sfp, dfp)
			if err != nil {
				log.Println(err)
			}
		}

	}
	return
}

// A struct for returning custom error messages
type CustomError struct {
	What string
}

// Returns the error message defined in What as a string
func (e *CustomError) Error() string {
	return e.What
}


// Render template or fail
func (d *Descriptor) mustTemplate(field *string) {
	t, err := template.New("").Parse(*field)
	if err != nil {
		panic("Failed parse templates in " + (*field) + ": " + err.Error())
	}
	buf := new(bytes.Buffer)
	err = t.Execute(buf, (*d))
	if err != nil {
		panic("Failed execute templates in " + (*field) + ": " + err.Error())
	}
	*field = buf.String()
}


func mustTemplate(pattern string, params interface{}) string {
	t, err := template.New("").Parse(pattern)
	if err != nil {
		panic("Failed parse templates in " + (pattern) + ": " + err.Error())
	}
	buf := new(bytes.Buffer)
	err = t.Execute(buf, params)
	if err != nil {
		panic("Failed execute templates in " + (pattern) + ": " + err.Error())
	}
	return buf.String()
}
