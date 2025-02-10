package gen

import (
	"eptablegenerator/table/config"
	"eptablegenerator/table/xlsx"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type sheetType interface {
	GetSheetName() string
	Generate() (string, []string, error)
}

type defaultSheetType struct {
	SheetName string
	Data      *[][]string
}

func (d *defaultSheetType) GetSheetName() string {
	return d.SheetName
}

type structType struct {
	defaultSheetType
}

func (s *structType) Generate() (string, []string, error) {
	var content string
	forwardContent := []string{}

	return content, forwardContent, nil
}

type enumType struct {
	defaultSheetType
}

func (e *enumType) Generate() (string, []string, error) {
	var content string
	forwardContent := []string{}

	return content, forwardContent, nil
}

// type headerType struct {
// 	Name string
// 	Type string
// }

func GenerateUE(c *config.Config) error {
	if c == nil {
		return errors.New("config is nil")
	}

	var files []string
	err := filepath.WalkDir(c.SourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && filepath.Ext(d.Name()) == ".xlsx" {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return err
	}

	m := map[string][]sheetType{}
	for _, file := range files {
		x := xlsx.NewXLSX(file)
		fileName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		for sheetName, data := range x.Data {
			if strings.HasPrefix(sheetName, "!") {
				st := &structType{defaultSheetType{
					SheetName: sheetName[1:],
					Data:      &data,
				}}
				m[fileName] = append(m[fileName], st)

			} else if strings.HasPrefix(sheetName, "@") {
				et := &enumType{defaultSheetType{
					SheetName: sheetName[1:],
					Data:      &data,
				}}
				m[fileName] = append(m[fileName], et)
			}
		}
	}

	for key := range m {
		sort.Slice(m[key], func(i, j int) bool {
			_, isEnumi := m[key][i].(*enumType)
			_, isEnumj := m[key][j].(*enumType)
			return isEnumi && !isEnumj
		})
	}

	errs := []error{}
	for fileName, sheets := range m {
		var preContent string
		forwardContent := map[string]any{}
		var content string

		preContent += "// 이 파일은 자동으로 생성된 파일입니다. 수동으로 수정하지 마세요.\n"
		preContent += "\n"
		preContent += "#pragma once\n"
		preContent += "\n"
		preContent += "#include \"CoreMinimal.h\""
		preContent += "\n"

		sheetErrs := []error{}
		for _, sheet := range sheets {
			c, p, err := sheet.Generate()
			if err != nil {
				sheetErrs = append(sheetErrs, err)
				break
			}
			content += c
			for _, v := range p {
				forwardContent[v] = nil
			}
		}
		if len(sheetErrs) > 0 {
			errs = append(errs, sheetErrs...)
			break
		}

		path := filepath.Join(c.DestDir, fileName+".h")
		h, err := os.Create(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		var result string
		result += preContent
		result += "\n"
		for key := range forwardContent {
			result += key + ";\n"
		}
		result += "\n"
		result += content

		if _, err := h.WriteString(result); err != nil {
			errs = append(errs, err)
			h.Close()
			continue
		}
		h.Close()
	}

	var r error
	for _, err := range errs {
		r = errors.Join(r, err)
	}

	return r
}
