package library

import "config"

type Ini struct {
	Config_path string
}

func (ini *Ini) Parse() map[string] map[string] interface{} {

	cfg, err := config.ReadDefault(ini.Config_path)
	if err != nil {
		return nil
	}

	sections := []string{"client", "mysql"}
	res := make(map[string] map[string] interface{})

	for _, tv := range sections {
		section, err := cfg.SectionOptions(tv)
		if err == nil {
			s1 := make(map[string] interface{})
			for _, v := range section {
				options, err := cfg.String(tv, v)
				if err == nil {
					s1[v] = options
				}
			}
			res[tv] = s1;
		}
	}

	return res;
}
