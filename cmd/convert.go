/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "convert Notion data",
	Long: `Convert Notion data, you can convert the exported 
csv of table to a series of Markdown files.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return checkParams(cmd.Flags())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		csv, _ := cmd.Flags().GetString("csv")
		out, _ := cmd.Flags().GetString("out")

		os.MkdirAll(out, os.ModePerm)

		err := csvToMarkdowns(csv, out)
		if err != nil {
			os.RemoveAll(out)
			return err
		}

		fmt.Println("convert success")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// convertCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// convertCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	convertCmd.Flags().String("csv", "", "csv filepath")
	convertCmd.Flags().String("out", "generated", "output folder")
}

func checkParams(fs *pflag.FlagSet) error {
	// 检查csv选项
	csv, _ := fs.GetString("csv")
	if len(csv) == 0 {
		return errors.New("invalid argument: --csv")
	}

	inf, err := os.Lstat(csv)
	if err != nil {
		return err
	}

	if inf.IsDir() {
		return errors.New("not valid csv")
	}

	// 检查out选项
	out, _ := fs.GetString("out")
	if len(out) == 0 {
		return errors.New("invalid argument: --out")
	}

	_, err = os.Lstat(out)
	if err == nil {
		return fmt.Errorf("%s already existed", out)
	}

	if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func csvToMarkdowns(csv, out string) error {
	recs, err := parseCSV(csv)
	if err != nil {
		return err
	}

	// generate markdown under out
	return generateMarkdowns(recs, out)
}

func parseCSV(f string) ([]Article, error) {
	dat, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("read csv err: %v", err)
	}

	r := csv.NewReader(bytes.NewReader(dat))
	r.Comma = ','
	r.Comment = '#'

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read records err: %v", err)
	}

	if len(records) == 0 {
		return nil, errors.New("read csv empty")
	}

	fields := records[0]
	for i, f := range fields {
		fields[i] = strings.TrimSpace(f)
	}

	recs := []Article{}
	for _, r := range records[1:] {
		rt := reflect.TypeOf(Article{})
		rv := reflect.New(rt).Elem()
		for i, f := range r {
			if len(f) == 0 {
				continue
			}
			if strings.Contains(f, "'") {
				f = strings.ReplaceAll(f, "'", "\"")
			}
			fd := rv.FieldByName(strings.TrimSpace(fields[i]))
			if !fd.IsValid() {
				fmt.Println("what the fuck: ", fields[i])
			}
			fd.SetString(f)
		}
		recs = append(recs, rv.Interface().(Article))
	}

	return recs, nil
}

func generateMarkdowns(recs []Article, out string) error {
	for _, r := range recs {
		err := generateMarkdown(r, out)
		if err != nil {
			return err
		}
	}
	return nil
}

var liquidTags = `
---
title: '{{.About}}'
score: '{{.Score}}'
tags: ['{{.Category}}']
author: '{{.Author}}'
publisher: '{{.Publisher}}'
status: '{{.Status}}'
link: '{{.Link}}'
---

# Let's Summarize

{{.Summary}}

# Source Analysis

{{.Source}}

# References
1. {{.Link}}
`

var tpl *template.Template

func init() {
	tpl = template.New("markdown")
	t, err := tpl.Parse(liquidTags)
	if err != nil {
		log.Fatal(err)
	}
	tpl = t
}

func generateMarkdown(r Article, out string) error {
	out = filepath.Join(out, r.About+".md")

	buf := &bytes.Buffer{}
	err := tpl.Execute(buf, r)
	if err != nil {
		return err
	}

	return os.WriteFile(out, buf.Bytes(), 0644)
}

type Article struct {
	Category  string
	About     string
	Status    string
	Score     string
	Link      string
	Author    string
	Publisher string
	Summary   string
	Source    string
}
