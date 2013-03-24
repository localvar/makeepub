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
		reKey     = regexp.MustCompile("^[ \t]*([^ \t=]+)[ \t]*=[ \t]*([^ \t]*)[ \t]*$")
	)

	section := "/"
	cfg := &Config{data: make(map[string]string)}

	for br := bufio.NewReader(reader); ; {
		line, _, e := br.ReadLine()
		if e != nil {
			if e == io.EOF {
				break
			}
			return nil, e
		}

		s := string(line)
		if reComment.MatchString(s) {
			continue
		}

		if m := reSection.FindStringSubmatch(s); m != nil {
			section = "/" + m[1]
			continue
		}

		if m := reKey.FindStringSubmatch(s); m != nil {
			k := strings.ToLower(section + "/" + m[1])
			cfg.data[k] = m[2]
		}
	}

	return cfg, nil
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
