package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	data map[string]string
}

func ParseIni(reader io.Reader) (*Config, error) {
	section, lastKey, cfg := "/", "", make(map[string]string)
	firstLine, scanner := true, bufio.NewScanner(reader)

	for scanner.Scan() {
		s := scanner.Bytes()
		if firstLine {
			s = removeUtf8Bom(s)
			firstLine = false
		}

		s = bytes.TrimSpace(s)
		if len(s) == 0 || s[0] == '#' { // empty or comment
			continue
		}

		if s[0] == '[' && s[len(s)-1] == ']' { // section
			s = bytes.TrimSpace(s[1 : len(s)-1])
			if len(s) >= 0 {
				section = "/" + string(bytes.ToLower(s))
			}
			continue
		}

		k, v := "", ""
		if i := bytes.IndexByte(s, '='); i != -1 {
			k = string(bytes.ToLower(bytes.TrimSpace(s[:i])))
			v = string(bytes.TrimSpace(s[i+1:]))
		}

		if len(k) > 0 {
			lastKey = section + "/" + k
			cfg[lastKey] = v
			continue
		} else if len(lastKey) == 0 {
			continue
		}

		c, lv := byte(128), cfg[lastKey]
		if len(lv) > 0 {
			c = lv[len(lv)-1]
		}

		if len(v) == 0 { // empty value means a new line
			cfg[lastKey] = lv + "\n"
		} else if c < 128 && c != '-' && v[0] < 128 { // need a white space?
			// not good enough, but should be ok in most cases
			cfg[lastKey] = lv + " " + v
		} else {
			cfg[lastKey] = lv + v
		}
	}

	if e := scanner.Err(); e != nil {
		return nil, e
	}

	return &Config{data: cfg}, nil
}

func OpenIniFile(path string) (*Config, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()

	return ParseIni(f)
}

func (cfg *Config) GetInt(path string, dflt int) int {
	path = strings.ToLower(path)
	if v, ok := cfg.data[path]; ok {
		if i, e := strconv.Atoi(v); e == nil {
			return i
		}
	}
	return dflt
}

func (cfg *Config) GetString(path string, dflt string) string {
	path = strings.ToLower(path)
	if v, ok := cfg.data[path]; ok {
		return v
	}
	return dflt
}

func (cfg *Config) GetBool(path string, dflt bool) bool {
	path = strings.ToLower(path)
	if v, ok := cfg.data[path]; ok {
		if b, e := strconv.ParseBool(v); e == nil {
			return b
		}
	}
	return dflt
}
