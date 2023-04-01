package config

import (
	"os"
	"regexp"

	"github.com/porridge/calendar-tracker/internal/core"
	"gopkg.in/yaml.v3"
)

type config struct {
	Categories []categoryConfig `yaml:"categories"`
}

type categoryConfig struct {
	Name  string        `yaml:"name"`
	Match []matchConfig `yaml:"match"`
}

type matchConfig struct {
	Regex string `yaml:"re"`
}

func Read(fileName string) ([]*core.Category, error) {
	c := &config{}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, c)
	if err != nil {
		return nil, err
	}
	var ret []*core.Category
	for _, cc := range c.Categories {
		var pp []*regexp.Regexp
		for _, p := range cc.Match {
			pp = append(pp, regexp.MustCompile(p.Regex))
		}
		ret = append(ret, &core.Category{
			Name:     core.CategoryName(cc.Name),
			Patterns: pp,
		})
	}
	return ret, nil
}
