// Package excel 读写有限大小的 XLSX 文本工作表，不执行公式或外部链接。
package excel

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

const maxExpanded = 20 * 1024 * 1024

// SafeText 防止表格软件把导出文本解释为公式。
func SafeText(s string) string {
	trimmed := strings.TrimLeft(s, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + s
	}
	if strings.HasPrefix(s, "\t") || strings.HasPrefix(s, "\r") {
		return "'" + s
	}
	return s
}

// Write 使用 Excelize 将所有字段写为文本，不产生公式。
func Write(rows [][]string) ([]byte, error) {
	book := excelize.NewFile()
	defer func() { _ = book.Close() }()
	size := 0
	for i, row := range rows {
		for j, value := range row {
			if j >= 26 {
				return nil, errors.New("列数超过限制")
			}
			size += len(value)
			if size > maxExpanded {
				return nil, errors.New("导出内容超过限制")
			}
			cell, err := excelize.CoordinatesToCellName(j+1, i+1)
			if err != nil {
				return nil, err
			}
			if err := book.SetCellStr("Sheet1", cell, SafeText(value)); err != nil {
				return nil, err
			}
		}
	}
	out, err := book.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

type richText struct {
	Text string `xml:"t"`
	Runs []struct {
		Text string `xml:"t"`
	} `xml:"r"`
}

func (t richText) text() string {
	out := t.Text
	for _, r := range t.Runs {
		out += r.Text
	}
	return out
}

type cell struct {
	Ref     string   `xml:"r,attr"`
	Type    string   `xml:"t,attr"`
	Value   string   `xml:"v"`
	Inline  richText `xml:"is"`
	Formula *string  `xml:"f"`
}
type row struct {
	Index int    `xml:"r,attr"`
	Cells []cell `xml:"c"`
}

// Read 读取 sheet1，限制 ZIP 总解压量、单元格长度和稀疏坐标，拒绝公式。
func Read(data []byte, maxRows int) ([][]string, error) {
	bad := errors.New("表格格式或大小不符合限制")
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(z.File) > 100 {
		return nil, bad
	}
	var expanded uint64
	files := map[string][]byte{}
	for _, f := range z.File {
		if f.UncompressedSize64 > maxExpanded-expanded {
			return nil, bad
		}
		expanded += f.UncompressedSize64
		if expanded > maxExpanded {
			return nil, bad
		}
		if f.Name != "xl/worksheets/sheet1.xml" && f.Name != "xl/sharedStrings.xml" {
			continue
		}
		if _, exists := files[f.Name]; exists {
			return nil, bad
		}
		r, e := f.Open()
		if e != nil {
			return nil, bad
		}
		raw, e := io.ReadAll(io.LimitReader(r, maxExpanded+1))
		closeErr := r.Close()
		if e != nil || closeErr != nil || len(raw) > maxExpanded {
			return nil, bad
		}
		files[f.Name] = raw
	}
	sheet, ok := files["xl/worksheets/sheet1.xml"]
	if !ok {
		return nil, bad
	}
	shared := []string{}
	if raw, ok := files["xl/sharedStrings.xml"]; ok {
		var table struct {
			Items []richText `xml:"si"`
		}
		if xml.Unmarshal(raw, &table) != nil {
			return nil, bad
		}
		if len(table.Items) > maxRows*26 {
			return nil, bad
		}
		for _, item := range table.Items {
			text := item.text()
			if len(text) > 4096 {
				return nil, bad
			}
			shared = append(shared, text)
		}
	}
	decoder := xml.NewDecoder(bytes.NewReader(sheet))
	out := [][]string{}
	for {
		tok, e := decoder.Token()
		if errors.Is(e, io.EOF) {
			break
		}
		if e != nil {
			return nil, bad
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "row" {
			continue
		}
		var r row
		if decoder.DecodeElement(&r, &start) != nil || len(out) >= maxRows || len(r.Cells) > 26 || r.Index > maxRows {
			return nil, bad
		}
		rowIndex := len(out) + 1
		if r.Index > 0 {
			rowIndex = r.Index
		}
		if rowIndex <= len(out) {
			return nil, bad
		}
		for len(out) < rowIndex-1 {
			out = append(out, make([]string, 26))
		}
		values := make([]string, 26)
		seen := map[int]bool{}
		for i, c := range r.Cells {
			if c.Formula != nil {
				return nil, errors.New("不接受公式单元格")
			}
			col := i
			if c.Ref != "" {
				letters := strings.TrimRight(c.Ref, "0123456789")
				if len(letters) != 1 || letters[0] < 'A' || letters[0] > 'Z' {
					return nil, bad
				}
				col = int(letters[0] - 'A')
				number, e := strconv.Atoi(strings.TrimPrefix(c.Ref, letters))
				if e != nil || number != rowIndex || number > maxRows {
					return nil, bad
				}
			}
			if seen[col] {
				return nil, bad
			}
			seen[col] = true
			value := c.Value
			switch c.Type {
			case "inlineStr":
				value = c.Inline.text()
			case "s":
				index, e := strconv.Atoi(c.Value)
				if e != nil || index < 0 || index >= len(shared) {
					return nil, bad
				}
				value = shared[index]
			case "", "str", "n", "b":
			default:
				return nil, bad
			}
			if len(value) > 4096 {
				return nil, bad
			}
			values[col] = value
		}
		out = append(out, values)
	}
	if len(out) < 1 {
		return nil, bad
	}
	book, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{RawCellValue: true, UnzipSizeLimit: maxExpanded, UnzipXMLSizeLimit: maxExpanded})
	if err != nil {
		return nil, bad
	}
	defer func() { _ = book.Close() }()
	if len(book.GetSheetList()) != 1 {
		return nil, errors.New("导入只接受单工作表")
	}
	return out, nil
}
