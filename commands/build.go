// Copyright 2018 Kevin Kenan
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy
// of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package commands

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevinkenan/cobra"
	"github.com/kevinkenan/subtext/core"
)

const (
	buildDesc = `Copies the contents from the specified to directory to the output directory,
processing subtext files as it goes.
`
)

func Build() (cmd *cobra.Command) {
	cmd = cobra.NewCommand("build")
	cmd.Short = "create a site"
	cmd.Long = buildDesc
	cmd.RunE = BuildRunE
	cmd.AddFlags(
		cobra.NewStringFlag("output", cobra.Opts().Abbr("o").Req(true).Desc("path to the output directory")),
		cobra.NewBoolFlag("recurse", cobra.Opts().Default(false).Desc("includes contents of subdirectories")),
		cobra.NewBoolFlag("reflow", cobra.Opts().Default(false).Desc("reflow paragraphs")),
		cobra.NewStringFlag("format", cobra.Opts().Desc("the output format")),
		cobra.NewStringSliceFlag("package-dir", cobra.Opts().Desc("path to a package directory. you may set this multiple times")),
		cobra.NewStringSliceFlag("packages", cobra.Opts().Abbr("p").Desc("macro package(s) to apply to input")))

	return
}

func BuildRunE(cmd *cobra.Command, args []string) (err error) {
	cobra.Log("beginning build cmd")
	cmd.SilenceUsage = true

	if len(args) == 0 {
		return fmt.Errorf("you must specify a source directory")
	}

	cobra.WithField("files", args).Log("processing")
	outdir := cobra.GetString("output")
	f := core.NewFolio()
	f.Cmd = cmd

	for _, pdir := range cobra.GetStringSlice("package-dir") {
		path := filepath.Clean(pdir)
		f.PkgSearchPaths = append(f.PkgSearchPaths, path)
	}

	f.Packages = cobra.GetStringSlice("packages")
	if len(f.Packages) > 0 {
		err = f.LoadPackages(f.Packages)
		if err != nil {
			return err
		}
	}

	for _, a := range args {
		err = copyDir(a, outdir, f)
		if err != nil {
			return err
		}
	}

	return nil
}

func copyDir(src, outdir string, folio *core.Folio) (err error) {
	src = filepath.Clean(src)
	outdir = filepath.Clean(outdir)

	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("copydir src: %s", err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(outdir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(outdir, srcInfo.Mode())
			if err != nil {
				return fmt.Errorf("unable to create output directory: %s", err)
			}
		} else {
			return fmt.Errorf("unable to read output directory: %s", err)
		}
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return fmt.Errorf("unable to read source directory: %s", err)
	}

	indexes := []string{}

	for _, entry := range entries {
		srcpath := filepath.Join(src, entry.Name())

		if entry.IsDir() {
			subdir := filepath.Join(outdir, entry.Name())
			err = copyDir(srcpath, subdir, folio)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			switch filepath.Ext(srcpath) {
			case ".stm":
				// skip
			case ".st":
				if strings.HasPrefix(filepath.Base(srcpath), "index.") {
					indexes = append(indexes, srcpath)
					continue
				}

				err = makeFile(srcpath, outdir, folio)
				if err != nil {
					return
				}
			default:
				err = copyFile(srcpath, outdir)
				if err != nil {
					return
				}
			}
		}
	}

	for _, i := range indexes {
		err = makeFile(i, outdir, folio)
		if err != nil {
			return
		}
	}

	return
}

func makeFile(src, outdir string, folio *core.Folio) (err error) {
	input, err := ioutil.ReadFile(src)
	if err != nil {
		return fmt.Errorf("makefile: %s", err)
	}

	srcname := filepath.Base(src)
	d := core.NewDoc(srcname, src)
	err = folio.AppendDoc(d)
	if err != nil {
		return
	}
	d.Text = string(input)
	// d.Plain = true

	output, err := d.Make()
	if err != nil {
		return
	}

	// d.Text = output
	// d.Plain = false
	// output, err = d.Make()
	// if err != nil {
	// 	return
	// }

	outname := d.OutputName
	if d.OutputName == "" {
		outname = fmt.Sprintf("%s.%s", strings.TrimSuffix(srcname, ".st"), d.Format)
	}

	dstpath, err := filepath.Abs(filepath.Join(outdir, outname))
	if err != nil {
		return fmt.Errorf("makefile: %s", err)
	}

	fo, err := os.Create(dstpath)
	if err != nil {
		return fmt.Errorf("makefile: %s", err)
	}
	defer func() {
		if e := fo.Close(); e != nil {
			err = e
		}
	}()

	_, err = fo.WriteString(output)
	if err != nil {
		return fmt.Errorf("makefile: %s", err)
	}

	err = fo.Sync()
	if err != nil {
		return fmt.Errorf("makefile: %s", err)
	}

	si, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("makefile: %s", err)
	}

	err = os.Chmod(dstpath, si.Mode())
	if err != nil {
		return fmt.Errorf("makefile: %s", err)
	}

	return
}

func copyFile(src, outdir string) (err error) {
	fname := filepath.Base(src)
	dst := filepath.Join(outdir, fname)

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copyfile: %s", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("copyfile: %s", err)
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("copyfile: %s", err)
	}

	err = out.Sync()
	if err != nil {
		return fmt.Errorf("copyfile: %s", err)
	}

	si, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("copyfile: %s", err)
	}

	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return fmt.Errorf("copyfile: %s", err)
	}

	return
}
