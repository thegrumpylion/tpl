package main

import (
	"bytes"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/spf13/cobra"
)

var genCmdArgs = struct {
	Template string
	Dir      string
}{}

func init() {
	genCmd.Flags().StringVarP(&genCmdArgs.Template, "template", "t", "", "template file")
	genCmd.MarkFlagRequired("template")
	genCmd.Flags().StringVarP(&genCmdArgs.Dir, "dir", "d", "", "directory to generate files in")
	rootCmd.AddCommand(genCmd)
}

var genCmd = &cobra.Command{
	Use:  "gen",
	RunE: runEgenCmd,
}

func runEgenCmd(cmd *cobra.Command, args []string) error {

	src := "templates/" + genCmdArgs.Template
	trgt := genCmdArgs.Dir
	modName := args[0]

	sprigFuncMap := sprig.GenericFuncMap()

	if err := os.MkdirAll(trgt, os.ModePerm); err != nil {
		return err
	}

	err := fs.WalkDir(tpls, src, func(path string, d fs.DirEntry, err error) error {

		if path == src {
			return nil
		}

		trgtName := path[len(src)+1:]
		// go embed limitation
		if trgtName == "go.mod.tpl" {
			trgtName = "go.mod"
		}

		nameTpl, err := template.New("").Delims("{{{", "}}}").Funcs(sprigFuncMap).Parse(trgtName)
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		err = nameTpl.Execute(&buf, tplContext{
			Name: modName,
			Org:  "thegrumpylion",
		})
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(filepath.Join(trgt, buf.String()), os.ModePerm)
		}

		fileTplContent, err := tpls.ReadFile(path)
		if err != nil {
			return err
		}

		fileTpl, err := template.New("").Delims("{{{", "}}}").Funcs(sprigFuncMap).Parse(string(fileTplContent))
		if err != nil {
			return err
		}

		f, err := os.Create(filepath.Join(trgt, buf.String()))
		if err != nil {
			return err
		}

		err = fileTpl.Execute(f, tplContext{
			Name: modName,
			Org:  "thegrumpylion",
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func getTemplateFolder(src string) (string, error) {
	u, err := url.Parse(src)
	if err != nil {
		// is url
		switch u.Scheme {

		}
	}

	src, isDir := checkPathExistence(src)
	if !isDir {
		// handle archive
		return handleArchive()
	}
}

func checkPathExistence(src string) (string, bool) {

	// check if the input string refers to a dir.
	fileInfo, err := os.Stat(src)
	if err == nil && !fileInfo.IsDir() {
		return src, nil
	}

	// check if the input string refers to an archive.
	for _, suffix := range archiveSuffixes {
		archivePath := src + suffix
		fileInfo, err := os.Stat(archivePath)
		if err == nil && !fileInfo.IsDir() {
			return handleArchive(archivePath)
		}
	}

	// Check if the input string refers to a file or directory in the $TPL_PATH.
	pathList := os.Getenv("TPL_PATH")
	paths := strings.Split(pathList, string(os.PathListSeparator))
	for _, path := range paths {
		fullPath := filepath.Join(path, src)
		fileInfo, err := os.Stat(fullPath)
		if err == nil && !fileInfo.IsDir() {
			return fullPath, nil
		}
		for _, suffix := range archiveSuffixes {
			arhivePath := fullPath + suffix
			fileInfo, err := os.Stat(arhivePath)
			if err == nil && !fileInfo.IsDir() {
				return handleArchive(arhivePath)
			}
		}
	}

	// Return an empty string and false if the file or directory is not found.
	return "", false
}
