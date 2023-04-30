package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
)

var genCmdArgs = struct {
	Template   string
	LeftDelim  string
	RightDelim string
}{}

func init() {
	genCmd.Flags().StringVarP(&genCmdArgs.Template, "template", "t", "", "template file")
	genCmd.MarkFlagRequired("template")
	genCmd.Flags().StringVarP(&genCmdArgs.LeftDelim, "left-delimiter", "l", "{{{", "left delimiter")
	genCmd.Flags().StringVarP(&genCmdArgs.RightDelim, "right-delimiter", "r", "}}}", "right delimiter")
	rootCmd.AddCommand(genCmd)
}

var genCmd = &cobra.Command{
	Use:  "gen",
	RunE: runEgenCmd,
	Args: cobra.ExactArgs(1),
}

func runEgenCmd(cmd *cobra.Command, args []string) error {

	u, err := url.Parse(genCmdArgs.Template)
	if err != nil {
		return err
	}

	if u.Host == "" {
		u.Host = "github.com"
	}

	if u.Scheme == "" {
		u.Scheme = "https"
	}

	src, err := cloneTemplate(u)
	if err != nil {
		return err
	}

	trgtDir := args[0]

	sprigFuncMap := sprig.GenericFuncMap()

	if err := os.MkdirAll(trgtDir, os.ModePerm); err != nil {
		return err
	}

	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {

		// ignore "." dir
		if path == src {
			return nil
		}

		// strip src path
		trgtName := path[len(src)+1:]

		// TODO:make this configurable
		// ignore .git dir
		if strings.HasPrefix(trgtName, ".git") {
			return nil
		}

		// create file name template
		nameTpl, err := template.New("").Delims(genCmdArgs.LeftDelim, genCmdArgs.RightDelim).Funcs(sprigFuncMap).Parse(trgtName)
		if err != nil {
			return err
		}

		// render file name
		var buf bytes.Buffer
		err = nameTpl.Execute(&buf, templateContext())
		if err != nil {
			return err
		}

		// create dir and return
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(trgtDir, buf.String()), os.ModePerm)
		}

		// read file template content
		fileTplContent, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		// parse file template
		fileTpl, err := template.New("").Delims(genCmdArgs.LeftDelim, genCmdArgs.RightDelim).Funcs(sprigFuncMap).Parse(string(fileTplContent))
		if err != nil {
			return err
		}

		// TODO: check if file exists and ask for overwrite
		// create file
		f, err := os.Create(filepath.Join(trgtDir, buf.String()))
		if err != nil {
			return err
		}

		// render file template
		err = fileTpl.Execute(f, templateContext())
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

func cloneTemplate(u *url.URL) (string, error) {

	hm, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	trgt := filepath.Join(hm, ".cache/tpl", u.Host, u.Path)

	// check if already cloned
	fi, err := os.Stat(trgt)
	if err == nil && fi.IsDir() {
		r, err := git.PlainOpen(trgt)
		if err != nil {
			return "", err
		}
		w, err := r.Worktree()
		if err != nil {
			return "", err
		}
		if err := w.Pull(&git.PullOptions{RemoteName: "origin"}); err != nil {
			if err != git.NoErrAlreadyUpToDate {
				return "", err
			}
		}
		return trgt, nil
	}

	if err := os.MkdirAll(trgt, os.ModePerm); err != nil {
		return "", err
	}

	fmt.Println(u.String())

	_, err = git.PlainClone(trgt, false, &git.CloneOptions{
		URL:      u.String(),
		Progress: os.Stdout,
	})
	if err != nil {
		return "", err
	}
	return trgt, nil
}

func templateContext() map[string]interface{} {
	return map[string]interface{}{
		"env": envMap(),
	}
}

func envMap() map[string]string {
	m := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		m[parts[0]] = parts[1]
	}
	fmt.Println(m)
	return m
}
