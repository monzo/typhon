package slog

import (
	"regexp"
	"strconv"
)

var formatterRe = regexp.MustCompile(`%` +
	`[\+\-# 0]*` + // Flags
	`(?:\d*\.|\[(\d+)\]\*\.)?(?:\d+|\[(\d+)\]\*)?` + // Width and precision
	`(?:\[(\d+)\])?` + // Argument index
	`[vTtbcdoOqxXUeEfFgGsp%]`, // Verb
)

func countFmtOperands(input string) int {
	count, point := 0, 0
	for _, match := range formatterRe.FindAllStringSubmatch(input, -1) {
		if match[0] == "%%" {
			// Deliberately match the regexp on %% (to prevent overlapping matches), but stop them here
			continue
		}

		for _, flag := range match[1:] {
			if flag == "" {
				continue
			} else if i, err := strconv.Atoi(flag); err == nil && i > 0 {
				point = i
				if point > count {
					count = point
				}
			}
		}
		if match[3] == "" {
			point++
		}
		if point > count {
			count = point
		}
	}
	return count
}
