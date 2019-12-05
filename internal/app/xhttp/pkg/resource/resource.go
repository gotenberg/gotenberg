package resource

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/thecodingmachine/gotenberg/internal/pkg/normalize"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xassert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

/*
TemporaryDirectory is the directory
where all the resources directory
are located.
*/
const TemporaryDirectory string = "tmp"

// Resource helps managing
// arguments and files for a conversion.
type Resource struct {
	logger        xlog.Logger
	dirPath       string
	customHeaders map[string][]string
	args          map[ArgKey]string
	files         map[string]file
}

// New creates a Resource where its files will
// be located in the given directory name.
func New(logger xlog.Logger, directoryName string) (Resource, error) {
	const op string = "resource.New"
	resolver := func() (string, error) {
		dirPath := fmt.Sprintf("%s/%s", TemporaryDirectory, directoryName)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return "", err
		}
		absDirPath, err := filepath.Abs(dirPath)
		if err != nil {
			return "", err
		}
		return absDirPath, nil
	}
	dirPath, err := resolver()
	if err != nil {
		return Resource{}, xerror.New(op, err)
	}
	logger.DebugfOp(op, "resource directory '%s' created", directoryName)
	return Resource{
		logger:        logger,
		dirPath:       dirPath,
		customHeaders: make(map[string][]string),
		args:          make(map[ArgKey]string),
		files:         make(map[string]file),
	}, nil
}

// Close removes the working directory of the
// Resource if it exists.
func (r Resource) Close() error {
	const op string = "resource.Resource.Close"
	if _, err := os.Stat(r.dirPath); os.IsNotExist(err) {
		r.logger.DebugfOp(op, "resource directory '%s' does not exist, nothing to remove", r.dirPath)
		return nil
	}
	if err := os.RemoveAll(r.dirPath); err != nil {
		return xerror.New(op, err)
	}
	r.logger.DebugfOp(op, "resource directory '%s' removed", r.dirPath)
	return nil
}

// WithCustomHeader add a new custom header to the Resource.
// Given key should be in canonical format.
func (r *Resource) WithCustomHeader(key string, value []string) {
	const op string = "resource.Resource.WithCustomHeader"
	if strings.Contains(key, RemoteURLCustomHeaderCanonicalBaseKey) ||
		strings.Contains(key, WebhookURLCustomHeaderCanonicalBaseKey) {
		r.customHeaders[key] = value
		r.logger.DebugfOp(op, "added '%s' with value '%s' to resource custom headers", key, value)
		return
	}
	r.logger.DebugfOp(op, "skipping '%s' as it is not a custom header...", key)
}

// WithArg add a new argument to the Resource.
func (r *Resource) WithArg(key ArgKey, value string) {
	const op string = "resource.Resource.WithArg"
	r.args[key] = value
	r.logger.DebugfOp(op, "added '%s' with value '%s' to resource args", key, value)
}

// WithFile add a new file to the Resource.
func (r *Resource) WithFile(filename string, in io.Reader) error {
	const op string = "resource.Resource.WithFile"
	resolver := func() error {
		// see https://github.com/thecodingmachine/gotenberg/issues/104.
		normalized, err := normalize.String(filename)
		if err != nil {
			return err
		}
		fpath := fmt.Sprintf("%s/%s", r.dirPath, normalized)
		file := file{fpath: fpath}
		if err := file.write(in); err != nil {
			return err
		}
		r.files[filename] = file
		r.logger.DebugfOp(op, "resource file '%s' created", filename)
		return nil
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// DirPath returns the directory path
// of the Resource.
func (r Resource) DirPath() string {
	return r.dirPath
}

// HasArg returns true if given key exists
// among the Resource and its value is not empty.
func (r Resource) HasArg(key ArgKey) bool {
	if v, ok := r.args[key]; ok {
		return v != ""
	}
	return false
}

/*
StringArg returns the value of the
argument identified by given key.

It works in the same manner as xassert.String.
*/
func (r Resource) StringArg(key ArgKey, defaultValue string, rules ...xassert.RuleString) (string, error) {
	const op string = "resource.Resource.StringArg"
	result, err := xassert.String(string(key), r.args[key], defaultValue, rules...)
	if err != nil {
		return result, xerror.New(op, err)
	}
	return result, nil
}

/*
Int64Arg returns the int64 representation of the
argument identified by given key.

It works in the same manner as xassert.Int64.
*/
func (r Resource) Int64Arg(key ArgKey, defaultValue int64, rules ...xassert.RuleInt64) (int64, error) {
	const op string = "resource.Resource.Int64Arg"
	result, err := xassert.Int64(string(key), r.args[key], defaultValue, rules...)
	if err != nil {
		return result, xerror.New(op, err)
	}
	return result, nil
}

/*
Float64Arg returns the float64 representation of the
argument identified by given key.

It works in the same manner as xassert.Float64.
*/
func (r Resource) Float64Arg(key ArgKey, defaultValue float64, rules ...xassert.RuleFloat64) (float64, error) {
	const op string = "resource.Resource.Float64Arg"
	result, err := xassert.Float64(string(key), r.args[key], defaultValue, rules...)
	if err != nil {
		return result, xerror.New(op, err)
	}
	return result, nil
}

/*
BoolArg returns the boolean representation of the
argument identified by given key.

It works in the same manner as xassert.Bool.
*/
func (r Resource) BoolArg(key ArgKey, defaultValue bool) (bool, error) {
	const op string = "resource.Resource.BoolArg"
	result, err := xassert.Bool(string(key), r.args[key], defaultValue)
	if err != nil {
		return result, xerror.New(op, err)
	}
	return result, nil
}

// Fpath returns the path of the given filename.
// This filename should exist whithin the Resource.
func (r Resource) Fpath(filename string) (string, error) {
	const op string = "resource.Resource.Fpath"
	file, ok := r.files[filename]
	if !ok {
		return "", xerror.Invalid(
			op,
			fmt.Sprintf("resource file '%s' does not exist", filename),
			nil,
		)
	}
	return file.fpath, nil
}

/*
Fpaths returns the paths of the files
having one of the given file extensions.

It should found at least one path.
*/
func (r Resource) Fpaths(exts ...string) ([]string, error) {
	const op string = "resource.Resource.Fpaths"
	var fpaths []string
	for filename, file := range r.files {
		for _, ext := range exts {
			if filepath.Ext(filename) == ext {
				fpaths = append(fpaths, file.fpath)
			}
		}
	}
	if len(fpaths) == 0 {
		return nil, xerror.Invalid(
			op,
			fmt.Sprintf("no resource file found for extensions '%v'", exts),
			nil,
		)
	}
	return fpaths, nil
}

/*
Fcontent returns the string content of the
given filename.

If filename does not exist within the Resource,
returns the default value.
*/
func (r Resource) Fcontent(filename, defaultValue string) (string, error) {
	const op string = "resource.Resource.Fcontent"
	file, ok := r.files[filename]
	if !ok {
		return defaultValue, nil
	}
	content, err := file.content()
	if err != nil {
		return "", xerror.New(op, err)
	}
	return content, nil
}
