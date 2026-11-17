package excel

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestXLSXRoundTripAndFormulaDefense(t *testing.T) {
	raw, err := Write([][]string{{"name", "text"}, {"中文 & <文本>", " =SUM(1,2)"}, {"x", "@SUM(A1:A2)"}})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := Read(raw, 1001)
	if err != nil || rows[1][0] != "中文 & <文本>" || rows[1][1] != "' =SUM(1,2)" || rows[2][1] != "'@SUM(A1:A2)" {
		t.Fatalf("表格文字或公式防护失效 %v", err)
	}
	if _, err := Read(raw, 2); err == nil {
		t.Fatal("行数限制失效")
	}
	if _, err := Read([]byte("invalid"), 1001); err == nil {
		t.Fatal("接受了非 XLSX 内容")
	}
}
func TestXLSXRejectsFormulaAndSparseColumns(t *testing.T) {
	for _, sheet := range []string{`<worksheet><sheetData><row><c r="A1"><f>1+1</f><v>2</v></c></row></sheetData></worksheet>`, `<worksheet><sheetData><row><c r="AA1" t="inlineStr"><is><t>x</t></is></c></row></sheetData></worksheet>`, `<worksheet><sheetData><row r="1000000"><c r="A1000000"><v>1</v></c></row></sheetData></worksheet>`} {
		var raw bytes.Buffer
		w := zip.NewWriter(&raw)
		f, err := w.Create("xl/worksheets/sheet1.xml")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(f, strings.NewReader(sheet)); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := Read(raw.Bytes(), 1001); err == nil {
			t.Fatal("危险表格未拒绝")
		}
	}
}
