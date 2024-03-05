package tool_gen_code

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"log"
	"os"
	"strings"
	"text/template"
	"unicode"
)

type Variable struct {
	Name     string
	VType    string
	JsonName string
	Comment  string
}

func UnderscoreToUpperCamelCase(s string) string {
	s = strings.Replace(s, "_", " ", -1)
	s = strings.Title(s)
	return strings.Replace(s, " ", "", -1)
}

func CamelCaseToUnderscore(s string) string {
	var output []rune
	for i, r := range s {
		if i == 0 {
			output = append(output, unicode.ToLower(r))
		} else {
			if unicode.IsUpper(r) {
				output = append(output, '_')
			}

			output = append(output, unicode.ToLower(r))
		}
	}
	return string(output)
}

func Gen() error {
	tplModel, err := os.ReadFile("./tool_gen_code/code_template.tpl")
	if err != nil {
		return err
	}
	parseChart, err := excelize.OpenFile("./tool_gen_code/chart.xlsx")
	if err != nil {
		return err
	}
	outputPath := "./tool_gen_code/result/"
	chartSheet := "Sheet1"

	data := make(map[string][]*Variable)
	for i := 2; ; i++ {
		keyName, err := parseChart.GetCellValue(chartSheet, fmt.Sprintf("B%d", i))
		if err != nil {
			return err
		}
		if keyName == "" {
			break
		}
		structName, err := parseChart.GetCellValue(chartSheet, fmt.Sprintf("A%d", i))
		if err != nil {
			return err
		}
		valueType, err := parseChart.GetCellValue(chartSheet, fmt.Sprintf("C%d", i))
		if err != nil {
			return err
		}
		comment, err := parseChart.GetCellValue(chartSheet, fmt.Sprintf("D%d", i))
		if err != nil {
			return err
		}

		data[structName] = append(data[structName], &Variable{
			Name:     UnderscoreToUpperCamelCase(keyName),
			VType:    valueType,
			JsonName: keyName, // Name变量名可根据代码规范调整，JsonName这里别做任何转化，这是他们那边要的效果
			Comment:  comment,
		})
	}
	log.Println(data)
	for structName, kv := range data {
		_, err = os.Stat(outputPath)
		if err != nil {
			err = os.Mkdir(outputPath, os.ModeDir)
			if err != nil {
				return err
			}
		}
		var writeFile *os.File
		_, err = os.Stat(outputPath + structName + ".gen.go")
		if err == nil {
			os.Remove(outputPath + structName + ".gen.go")
		}
		writeFile, err = os.Create(outputPath + structName + ".gen.go")
		if err != nil {
			return nil
		}
		var Fills struct {
			PackageName string
			StructName  string
			KV          []*Variable
		}
		Fills.PackageName = "result"
		Fills.StructName = UnderscoreToUpperCamelCase(structName)
		Fills.KV = kv
		tmpl, _ := template.New("test").Parse(string(tplModel))
		err = tmpl.Execute(writeFile, Fills)
		if err != nil {
			return err
		}
		err = writeFile.Close()
		if err != nil {
			return err
		}
		log.Printf("output success to result.gen.go")
	}
	return nil
}
