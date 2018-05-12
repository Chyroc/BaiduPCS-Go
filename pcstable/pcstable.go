// Package pcstable 格式化表格包
package pcstable

import (
	"io"

	"github.com/olekukonko/tablewriter"
)

// PCSTable 封装 tablewriter.Table
type PCSTable struct {
	*tablewriter.Table
}

// NewTable 预设了一些配置
func NewTable(wt io.Writer) PCSTable {
	tb := tablewriter.NewWriter(wt)
	tb.SetAutoWrapText(false)
	tb.SetBorder(false)
	tb.SetHeaderLine(false)
	tb.SetColumnSeparator("")
	return PCSTable{tb}
}
