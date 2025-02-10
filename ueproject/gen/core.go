package gen

import (
	"eptablegenerator/table/config"
	"eptablegenerator/table/xlsx"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type sheetType interface {
	GetSheetName() string
	Generate() (string, []string, []string, error)
}

type defaultSheetType struct {
	ProjectName string
	SheetName   string
	Data        *[][]string
}

func (d *defaultSheetType) GetSheetName() string {
	return d.SheetName
}

type structType struct {
	defaultSheetType
}

func (s *structType) Generate() (string, []string, []string, error) {
	var content string
	forwardContent := []string{}
	include := []string{}

	if len(*s.Data) < 2 {
		return content, forwardContent, include, errors.New("data is not enough")
	}

	// NOTE. 첫번 째 헤더 이름, 두번 째 값 타입
	header := (*s.Data)[0]
	types := (*s.Data)[1]

	if len(header) != len(types) {
		return content, forwardContent, include, errors.New("header and types are not matched")
	}

	variables := []string{}
	for _, v := range types {
		switch {
		case v == "bool":
			variables = append(variables, "bool")

		case v == "int32":
			variables = append(variables, "int32")

		case v == "int64":
			variables = append(variables, "int64")

		case v == "float32":
			variables = append(variables, "float")

		case v == "float64":
			variables = append(variables, "double")

		case v == "FString":
			variables = append(variables, "FString")

		case v == "FText":
			variables = append(variables, "FText")

		case strings.HasPrefix(v, "TArray<") && strings.HasSuffix(v, ">"):
			variables = append(variables, "TArray<"+v[7:len(v)-1]+">")

		case strings.HasPrefix(v, "TMap<") && strings.HasSuffix(v, ">"):
			variables = append(variables, "TMap<"+v[5:len(v)-1]+">")

		case strings.HasPrefix(v, "TSet<") && strings.HasSuffix(v, ">"):
			variables = append(variables, "TSet<"+v[5:len(v)-1]+">")

		case strings.HasPrefix(v, "Enum<") && strings.HasSuffix(v, ">"):
			variables = append(variables, v[5:len(v)-1])
			forwardContent = append(forwardContent, "enum class "+v[5:len(v)-1]+" : uint8")

		case strings.HasPrefix(v, "Class<") && strings.HasSuffix(v, ">"):
			variables = append(variables, "TSoftClassPtr<"+v[6:len(v)-1]+">")
			forwardContent = append(forwardContent, "class "+v[6:len(v)-1])

		case strings.HasPrefix(v, "Asset<") && strings.HasSuffix(v, ">"):
			variables = append(variables, "TSoftObjectPtr<"+v[6:len(v)-1]+">")
			forwardContent = append(forwardContent, "class "+v[6:len(v)-1])

		default:
			variables = append(variables, "")
		}
	}

	include = append(include, "Engine/DataTable.h")

	projectName := strings.ToUpper(s.ProjectName)
	if projectName != "" {
		projectName = fmt.Sprintf("%s_API ", projectName)
	}

	content += "USTRUCT(BlueprintType)\n"
	content += fmt.Sprintf("struct %sF%s : public FTableRowBase\n", projectName, s.SheetName)
	content += "{\n"
	content += "\tGENERATED_BODY()\n"
	content += "\n"

	duplicate := map[string]any{}
	for i, name := range header {
		v := variables[i]
		if v == "" || name == "" {
			continue
		}

		if _, ok := duplicate[name]; ok {
			continue
		}
		duplicate[name] = nil

		content += "\tUPROPERTY(EditAnywhere, BlueprintReadWrite)\n"
		content += fmt.Sprintf("\t%s %s;\n", v, name)
		content += "\n"
	}

	content += "}\n"
	content += "\n"

	return content, forwardContent, include, nil
}

type enumType struct {
	defaultSheetType
}

func (e *enumType) Generate() (string, []string, []string, error) {
	var content string
	forwardContent := []string{}
	include := []string{}

	if len(*e.Data) < 2 {
		return content, forwardContent, include, errors.New("data is not enough")
	}

	values := []string{}
	for _, data := range (*e.Data)[2:] {
		if len(data) < 2 {
			continue
		}

		value, err := strconv.Atoi(data[0])
		name := data[1]

		var displayName string
		if len(data) >= 3 {
			displayName = data[2]
		}

		if err != nil {
			println("value is not int: " + e.SheetName)
			continue
		}

		if name == "" {
			println("name is empty: " + e.SheetName)
			continue
		}

		r := fmt.Sprintf("\t%s = %d", name, value)
		if displayName != "" {
			r += fmt.Sprintf(" UMETA(DisplayName = \"%s\")", displayName)
		}

		values = append(values, r)
	}

	include = append(include, "Misc/EnumRange.h")

	content += "UENUM(BlueprintType)\n"
	content += fmt.Sprintf("enum class %s : uint8", e.SheetName)
	content += "{\n"

	for _, v := range values {
		content += v + ",\n"
	}

	content += "\tMax UMETA(Hidden)\n"
	content += "}\n"
	content += "\n"
	content += fmt.Sprintf("ENUM_RANGE_BY_COUNT(%s, %s::Max)\n", e.SheetName, e.SheetName)
	content += "\n"

	return content, forwardContent, include, nil
}

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
				// NOTE. 첫번 째 헤더 이름, 두번 째 값 타입
				sliceData := data[:2]
				st := &structType{defaultSheetType{
					ProjectName: c.ProjectName,
					SheetName:   sheetName[1:],
					Data:        &sliceData,
				}}
				m[fileName] = append(m[fileName], st)

			} else if strings.HasPrefix(sheetName, "@") {
				et := &enumType{defaultSheetType{
					ProjectName: c.ProjectName,
					SheetName:   sheetName[1:],
					Data:        &data,
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
	docs := map[string]string{}
	for fileName, sheets := range m {
		var preContent string
		include := map[string]any{}
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
			c, p, i, err := sheet.Generate()
			if err != nil {
				sheetErrs = append(sheetErrs, err)
				break
			}
			content += c
			for _, v := range i {
				include[v] = nil
			}
			for _, v := range p {
				forwardContent[v] = nil
			}
		}
		if len(sheetErrs) > 0 {
			errs = append(errs, sheetErrs...)
			break
		}

		var result string
		result += preContent
		result += "\n"
		for key := range include {
			result += fmt.Sprintf("#include \"%s\"\n", key)
		}
		result += fmt.Sprintf("#include \"%s.generated.h\"\n", fileName)

		for key := range forwardContent {
			result += key + ";\n"
		}
		result += "\n"
		result += content

		docs[fileName] = result

	}

	var r error
	for _, err := range errs {
		r = errors.Join(r, err)
	}

	if r == nil {
		for fileName, doc := range docs {
			path := filepath.Join(c.DestDir, fileName+".h")
			h, err := os.Create(path)
			if err != nil {
				r = errors.Join(r, err)
				continue
			}

			if _, err := h.WriteString(doc); err != nil {
				errs = append(errs, err)
				h.Close()
				continue
			}
			h.Close()
		}
	}

	return r
}
