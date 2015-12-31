package lib

import (
	"fmt"
	"strings"
	"syscall"
	"unicode"
	"unsafe"
)

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func getWidth() uint {
	ws := &winsize{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		panic(errno)
	}
	return uint(ws.Col)
}

func Format(file_names []string) string {
	term_width := int(getWidth())
	if term_width <= 0 {
		term_width = 80
	}
	total_width := 0
	tmp_col_length := []int{term_width}
	var col_length []int
	if length := maxStringLength(file_names); length > term_width {
		col_length = append(col_length, length)
	} else {
		for col := 2; total_width <= term_width; col++ {
			row := rows(len(file_names), col)
			if row*(col-1) > len(file_names) {
				continue
			}
			col_length = make([]int, len(tmp_col_length))
			copy(col_length, tmp_col_length)
			tmp_col_length = make([]int, col)
			total_width = 0
			if row == 0 {
				break
			}
			for i := 0; i < col-1; i++ {
				tmp_col_length[i] = maxStringLength(file_names[i*row:(i+1)*row]) + 2
				total_width += tmp_col_length[i] + 2
			}
			tmp_col_length[len(tmp_col_length)-1] = maxStringLength(file_names[(col-1)*row:len(file_names)]) + 2
			total_width += tmp_col_length[len(tmp_col_length)-1] + 2
		}
	}
	col := len(col_length) //Adjust column
	var column string
	var columns []string
	length := col_length[0]
	row := rows(len(file_names), col)
	col_idx := 0
	for idx, file_name := range file_names {
		if idx%row == 0 && idx != 0 {
			columns = append(columns, column)
			column = ""
			col_idx++
			length = col_length[col_idx]
		}
		space_length := length - calcStringWidth(file_name)
		if space_length < 0 {
			fmt.Println(file_name)
			fmt.Println(length)
			fmt.Println(len(file_name))
		}
		column = column + file_name + strings.Repeat(" ", space_length) + "\n"
	}
	columns = append(columns, column)
	left := columns[0]
	for _, c := range columns[1:] {
		left = combineStr(left, "\n", c)
	}
	return left
}
func rows(items, col int) int {
	if items/col == 0 {
		return 0
	} else if items%col == 0 {
		return items / col
	} else {
		return items/col + 1
	}
}

func maxStringLength(strlist []string) int {
	max := 0
	for _, str := range strlist {
		if calcStringWidth(str) > max {
			max = len(str)
		}
	}
	return max
}

func calcStringWidth(s string) int {
	width := 0
	for _, c := range s {
		if unicode.In(c, unicode.Hangul, unicode.Katakana, unicode.Hiragana, unicode.Han) {
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

func combineStr(left string, sep string, right string) string {
	left_lines := strings.Split(left, sep)
	right_lines := strings.Split(right, sep)
	var min int
	if len(left_lines) < len(right_lines) {
		min = len(left_lines)
	} else {
		min = len(right_lines)
	}
	for i := 0; i < min; i++ {
		left_lines[i] = left_lines[i] + right_lines[i]
	}
	return strings.Join(left_lines, sep)
}
