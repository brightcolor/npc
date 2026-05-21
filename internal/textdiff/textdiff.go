package textdiff

import (
	"fmt"
	"strings"
)

func Unified(nameA, a, nameB, b string) string {
	if a == b {
		return fmt.Sprintf("%s and %s are identical\n", nameA, nameB)
	}
	left := strings.Split(a, "\n")
	right := strings.Split(b, "\n")
	var out strings.Builder
	out.WriteString("--- " + nameA + "\n")
	out.WriteString("+++ " + nameB + "\n")
	max := len(left)
	if len(right) > max {
		max = len(right)
	}
	for i := 0; i < max; i++ {
		var l, r string
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		if l == r {
			continue
		}
		out.WriteString(fmt.Sprintf("@@ line %d @@\n", i+1))
		if i < len(left) {
			out.WriteString("-" + l + "\n")
		}
		if i < len(right) {
			out.WriteString("+" + r + "\n")
		}
	}
	return out.String()
}
