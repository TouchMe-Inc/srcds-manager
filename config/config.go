package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Root struct {
	Project  string   `yaml:"project"`
	Path     string   `yaml:"path"`
	Ip       string   `yaml:"ip"`
	Params   string   `yaml:"params"`
	Map      string   `yaml:"map"`
	Niceness *int     `yaml:"niceness,omitempty"`
	Idler    Idler    `yaml:"idler"`
	Servers  []Server `yaml:"servers"`
}

type Server struct {
	Path   string `yaml:"path"`
	Number int    `yaml:"number"`
	Ip     string `yaml:"ip"`
	Port   int    `yaml:"port"`
	// Deprecated: use Cores. Single CPU index, kept for backward compatibility.
	Core *int `yaml:"core,omitempty"`
	// Cores is a list of CPU indexes to pin the server to with `taskset -c`.
	// For SMT CPUs pass both siblings of a physical core (see
	// /sys/devices/system/cpu/cpu*/topology/thread_siblings_list),
	// e.g. [0, 1]. Empty means no pinning.
	Cores     []int  `yaml:"cores,omitempty"`
	Niceness  *int   `yaml:"niceness,omitempty"`
	Map       string `yaml:"map"`
	Params    string `yaml:"params"`
	ParamsAdd string `yaml:"params_add"`
	Idler     Idler  `yaml:"idler"`
}

type Idler struct {
	Enabled  *bool `yaml:"enabled,omitempty"`
	Niceness *int  `yaml:"niceness,omitempty"`
}

// Path current config file path
var Path = GetDefaultLocation()

// Value current config value
var Value = Root{}

var defaults = struct {
	Project       string
	Ip            string
	Niceness      int
	IdlerEnabled  bool
	IdlerNiceness int
}{
	Project:       "srcds-server",
	Ip:            "127.0.0.1",
	Niceness:      0,
	IdlerEnabled:  false,
	IdlerNiceness: -20,
}

// GetDefaultLocation returns default location and name for the config file (~/srcds.yaml)
func GetDefaultLocation() string {
	homedir, _ := os.LookupEnv("HOME")
	return filepath.Join(homedir, "/srcds.yaml")
}

// Read reads and validates config file from path
func Read() error {
	Reset()
	_, err := os.Stat(Path)
	if err != nil {
		return err
	}
	f, err := os.ReadFile(Path)
	if err != nil {
		return err
	}
	cfg := Root{}
	err = yaml.Unmarshal(f, &cfg)
	if err != nil {
		return err
	}

	// ВАЖНО: Расставляем номера серверам ДО вызова validate,
	// иначе валидатор увидит нули и выдаст "servers #0 and #0 share core..."
	for i := range cfg.Servers {
		if cfg.Servers[i].Number == 0 {
			cfg.Servers[i].Number = i + 1
		}
	}

	err = validate(Path, cfg)
	if err != nil {
		return err
	}

	Value = *setDefaults(&cfg)
	return nil
}

func ReadOrThrow() {
	err := Read()
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}

// Reset resets Value to its unread state
func Reset() {
	Value = Root{}
	setDefaults(&Value)
}

func validate(path string, cfg Root) error {
	if len(cfg.Servers) == 0 {
		return fmt.Errorf("config file %q must include at least one server", path)
	}

	seenPorts := map[int]bool{}
	seenNumbers := map[int]bool{}

	for i, server := range cfg.Servers {
		if server.Port == 0 {
			return fmt.Errorf("config file %q must have a `port` defined in `servers[%d]`", path, i)
		}

		if seenPorts[server.Port] {
			return fmt.Errorf("config file %q: duplicate port %d", path, server.Port)
		}
		seenPorts[server.Port] = true

		if seenNumbers[server.Number] {
			return fmt.Errorf("config file %q: duplicate server number %d (check `number` fields or order)", path, server.Number)
		}
		seenNumbers[server.Number] = true

		if server.Path == "" && cfg.Path == "" {
			return fmt.Errorf("config file %q: `servers[%d]` has no `path`, and top-level `path` is empty", path, i)
		}
	}
	return validateCoreOverlap(path, cfg)
}

// physicalCoreOf returns a stable physical-core id for a CPU index,
func physicalCoreOf(cpu int) int {
	b, err := os.ReadFile(fmt.Sprintf("/sys/devices/system/cpu/cpu%d/topology/thread_siblings_list", cpu))
	if err != nil {
		return cpu / 2
	}
	fields := strings.FieldsFunc(strings.TrimSpace(string(b)), func(r rune) bool {
		return r == ',' || r == '-'
	})
	if len(fields) == 0 {
		return cpu / 2
	}
	first, err := strconv.Atoi(fields[0])
	if err != nil {
		return cpu / 2
	}
	return first
}

// validateCoreOverlap ensures that no two servers are pinned to CPUs of the same physical core.
func validateCoreOverlap(path string, cfg Root) error {
	type owner struct {
		server int
		cores  []int
	}
	seen := map[int]owner{} // physical core -> owner

	for _, s := range cfg.Servers {
		cores := s.EffectiveCores()
		if len(cores) == 0 {
			continue
		}

		serverPhysCores := map[int]bool{}

		for _, c := range cores {
			phys := physicalCoreOf(c)
			serverPhysCores[phys] = true
		}

		for phys := range serverPhysCores {
			if o, ok := seen[phys]; ok && o.server != s.Number {
				return fmt.Errorf(
					"config file %q: servers #%d and #%d share physical core %d (cores %v vs %v); "+
						"use one whole physical core per server, e.g. cores: [0,1], [2,3], [4,5], [6,7]",
					path, o.server, s.Number, phys, o.cores, cores,
				)
			}

			seen[phys] = owner{server: s.Number, cores: cores}
		}
	}
	return nil
}

func setDefaults(cfg *Root) *Root {
	if cfg.Project == "" {
		cfg.Project = defaults.Project
	}
	if cfg.Ip == "" {
		cfg.Ip = defaults.Ip
	}
	if cfg.Idler.Enabled == nil {
		v := defaults.IdlerEnabled
		cfg.Idler.Enabled = &v
	}
	if cfg.Idler.Niceness == nil {
		v := defaults.IdlerNiceness
		cfg.Idler.Niceness = &v
	}
	if cfg.Niceness == nil {
		v := defaults.Niceness
		cfg.Niceness = &v
	}
	for i, server := range cfg.Servers {
		if server.Path == "" {
			server.Path = cfg.Path
		}
		if server.Ip == "" {
			server.Ip = cfg.Ip
		}
		if server.Map == "" {
			server.Map = cfg.Map
		}
		if server.Params == "" {
			server.Params = cfg.Params
		}
		if server.Number == 0 {
			server.Number = i + 1
		}
		if server.Niceness == nil {
			v := *cfg.Niceness
			server.Niceness = &v
		}
		if server.Idler.Enabled == nil {
			v := *cfg.Idler.Enabled
			server.Idler.Enabled = &v
		}
		if server.Idler.Niceness == nil {
			v := *cfg.Idler.Niceness
			server.Idler.Niceness = &v
		}
		cfg.Servers[i] = server
	}

	return cfg
}

func (s Server) EffectiveCores() []int {
	if len(s.Cores) > 0 {
		return s.Cores
	}
	if s.Core != nil {
		return []int{*s.Core}
	}
	return nil
}
