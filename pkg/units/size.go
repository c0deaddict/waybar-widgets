package units

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	sizeUnits  = []string{"", "k", "M", "G", "T", "P", "E", "Z", "Y"}
	sizeRegexp = regexp.MustCompile(`^(\d+)([a-zA-Z])??[bB]?$`)
)

func FromHumanSize(value string) (uint64, error) {
	m := sizeRegexp.FindStringSubmatch(value)
	if m == nil {
		return 0, fmt.Errorf("%s is not a valid size", value)
	}
	bytes, err := strconv.ParseUint(m[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s is not a valid size: %v", value, err)
	}
	for _, suffix := range sizeUnits {
		if strings.ToLower(m[2]) == strings.ToLower(suffix) {
			return bytes, nil
		}
		bytes *= 1024
	}
	return 0, fmt.Errorf("%s is not a valid size: unknown unit %s", value, m[2])
}

func HumanSizeExact(size uint64) string {
	unit := 0
	for size > 0 && size%1024 == 0 && unit < len(sizeUnits)-1 {
		size /= 1024
		unit += 1
	}
	return strconv.FormatUint(size, 10) + sizeUnits[unit] + "B"
}

func HumanSize(size uint64) string {
	unit := 0
	s := float64(size)
	for s > 1024 && unit < len(sizeUnits)-1 {
		s /= 1024
		unit += 1
	}
	return fmt.Sprintf("%.1f%sB", s, sizeUnits[unit])
}
