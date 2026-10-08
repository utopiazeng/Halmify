package controllers

import (
	"archive/zip"
	"fmt"
	"html"
	"io"
	"sort"
	"strings"
	"time"
)

type excelTable struct {
	Title   string
	Headers []string
	Rows    [][]interface{}
}

func writeExcelHTML(w io.Writer, tables ...excelTable) error {
	builder := &strings.Builder{}
	builder.WriteString(`<!doctype html><html><head><meta charset="UTF-8"><style>`)
	builder.WriteString(`body{font-family:Arial,"Microsoft YaHei",sans-serif;}table{border-collapse:collapse;margin-bottom:18px;}caption{font-weight:700;text-align:left;padding:8px 0;}th,td{border:1px solid #d9e2ef;padding:6px 10px;mso-number-format:"\@";}th{background:#eef4ff;color:#172033;}`)
	builder.WriteString(`</style></head><body>`)
	for _, table := range tables {
		builder.WriteString("<table>")
		if strings.TrimSpace(table.Title) != "" {
			builder.WriteString("<caption>")
			builder.WriteString(html.EscapeString(table.Title))
			builder.WriteString("</caption>")
		}
		builder.WriteString("<thead><tr>")
		for _, header := range table.Headers {
			builder.WriteString("<th>")
			builder.WriteString(html.EscapeString(header))
			builder.WriteString("</th>")
		}
		builder.WriteString("</tr></thead><tbody>")
		for _, row := range table.Rows {
			builder.WriteString("<tr>")
			for _, cell := range row {
				builder.WriteString("<td>")
				builder.WriteString(html.EscapeString(excelCellText(cell)))
				builder.WriteString("</td>")
			}
			builder.WriteString("</tr>")
		}
		builder.WriteString("</tbody></table>")
	}
	builder.WriteString("</body></html>")
	_, err := io.WriteString(w, builder.String())
	return err
}

func writeExcelHTMLZipFile(zipWriter *zip.Writer, name string, tables ...excelTable) error {
	file, err := zipWriter.Create(name)
	if err != nil {
		return err
	}
	return writeExcelHTML(file, tables...)
}

func excelCellText(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case time.Time:
		if typed.IsZero() {
			return ""
		}
		return typed.Format("2006-01-02 15:04:05")
	case *time.Time:
		if typed == nil || typed.IsZero() {
			return ""
		}
		return typed.Format("2006-01-02 15:04:05")
	case fmt.Stringer:
		return typed.String()
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func flattenMapRows(prefix string, value interface{}) [][]interface{} {
	rows := [][]interface{}{}
	switch typed := value.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			name := key
			if prefix != "" {
				name = prefix + "." + key
			}
			rows = append(rows, flattenMapRows(name, typed[key])...)
		}
	case []interface{}:
		if len(typed) == 0 {
			rows = append(rows, []interface{}{prefix, ""})
			break
		}
		for index, item := range typed {
			name := fmt.Sprintf("%s[%d]", prefix, index+1)
			rows = append(rows, flattenMapRows(name, item)...)
		}
	default:
		rows = append(rows, []interface{}{prefix, typed})
	}
	return rows
}
