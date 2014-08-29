package main

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Config struct {
	data map[string]string
}

func ParseIni(reader io.Reader) (*Config, error) {
	var (
		reComment = regexp.MustCompile("^[ \t]*#.*$")
		reSection = regexp.MustCompile("^[ \t]*\\[([^\\]]+)\\][ \t]*$")
		reKey     = regexp.MustCompile("^([^=]*)=[ \t]*([^ \t]*)[ \t]*$")
	)

	section, lastKey, cfg := "/", "", make(map[string]string)
	firstLine, scanner := true, bufio.NewScanner(reader)

	for scanner.Scan() {
		s := scanner.Bytes()
		if firstLine {
			s = removeUtf8Bom(s)
			firstLine = false
		}

		if reComment.Match(s) {
			continue
		}

		m := reSection.FindSubmatch(s)
		if m != nil {
			section = "/" + string(m[1])
			continue
		}

		if m = reKey.FindSubmatch(s); m == nil {
			continue
		}

		k := strings.ToLower(strings.TrimSpace(string(m[1])))
		if len(k) > 0 {
			lastKey = section + "/" + k
			cfg[lastKey] = string(m[2])
		} else if len(lastKey) > 0 {
			cfg[lastKey] = cfg[lastKey] + string(m[2])
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
