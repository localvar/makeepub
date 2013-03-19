package main

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
)

type IniFile struct {
	data map[string]string
}

func Open(path string) (*IniFile, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	br := bufio.NewReader(f)

	section := ""
	m := make(map[string]string)

	for {
		line, _, e := br.ReadLine()
		if e == io.EOF {
			break
		}

		s := strings.TrimSpace(string(line))
		if len(s) == 0 || s[0] == '#' {
			continue
		}

		if s[0] == '[' {
			if i := strings.Index(s, "]"); i != -1 {
				section = strings.ToLower(s[1:i])
			}
		} else {
			if i := strings.Index(s, "="); i != -1 {
				k := section + "." + strings.TrimSpace(strings.ToLower(s[0:i]))
				m[k] = strings.TrimSpace(s[i+1:])
			}
		}
	}

	return &IniFile{data: m}, nil
}

func (ini *IniFile) GetInt(section string, key string, dflt int) int {
	key = strings.ToLower(section + "." + key)
	if v, ok := ini.data[key]; ok {
		if i, e := strconv.Atoi(v); e == nil {
			return i
		}
	}
	return dflt
}

func (ini *IniFile) GetString(section, key, dflt string) string {
	key = strings.ToLower(section + "." + key)
	if v, ok := ini.data[key]; ok {
		return v
	}
	return dflt
}
