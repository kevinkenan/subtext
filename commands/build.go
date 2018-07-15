package commands

import (
	// "bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevinkenan/cobra"
	"github.com/kevinkenan/subtext/core"
)

func Build(cmd *cobra.Command, args []string) (err error) {
	cobra.Log("beginning build cmd")
	cmd.SilenceUsage = true

	if len(args) == 0 {
		return fmt.Errorf("you must specify a source directory")
	} else {
		cobra.WithField("files", args).Log("processing ")
		outf := cobra.GetString("output")
		for _, f := range args {
			err = copyDir(f, outf)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func copyDir(src string, dst string) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if err != nil {
		err = os.MkdirAll(dst, si.Mode())
		if err != nil {
			return
		}
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	indexes := []string{}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = copyDir(srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			switch filepath.Ext(srcPath) {
			case ".stm":
				// skip
			case ".st":
				if strings.HasPrefix(filepath.Base(srcPath), "index.") {
					indexes = append(indexes, srcPath)
					continue
				}

				err = makeFile(srcPath, dstPath)
				if err != nil {
					return
				}
			default:
				err = copyFile(srcPath, dstPath)
				if err != nil {
					return
				}
			}
		}
	}

	return
}

func makeFile(src, dst string) (err error) {
	input, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	f := core.NewFolio()
	d := core.NewDoc(src, filepath.Base(src))
	f.Append(d)
	d.Output = cobra.GetString("output")
	d.Text = string(input)

	output, err := d.Make()
	if err != nil {
		return err
	}

	fo, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if e := fo.Close(); e != nil {
			err = e
		}
	}()

	_, err = fo.WriteString(output)
	if err != nil {
		return
	}

	err = fo.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}

	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}

func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}

	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}
