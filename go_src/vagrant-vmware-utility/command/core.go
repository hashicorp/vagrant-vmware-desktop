package command

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/mitchellh/cli"
)

type ForceExit struct {
	ExitCode int
}

const ENV_VAR_PREFIX = "VAGRANT_VMWARE_UTILITY_"

func Commands(name string, ui cli.Ui) (cmds map[string]cli.CommandFactory) {
	cmds = map[string]cli.CommandFactory{
		"api":                  BuildRestApiCommand(name, ui),
		"certificate generate": BuildCertificateGenerateCommand(name, ui),
		"service install":      BuildServiceInstallCommand(name, ui),
		"service uninstall":    BuildServiceUninstallCommand(name, ui),
	}
	platformSpecificCommands(name, ui, cmds)
	return
}

// Base command stubs
type Command struct {
	DefaultConfig *Config
	Name          string
	Flags         *flag.FlagSet
	HelpText      string
	SynopsisText  string
	UI            cli.Ui
	flagdata      map[string]interface{}
	logger        hclog.Logger
}

type Config struct {
	Debug     bool
	Level     string
	LogFile   string
	LogAppend bool

	Pdebug     *bool   `hcl:"debug"`
	Plevel     *string `hcl:"level"`
	PlogFile   *string `hcl:"log_file"`
	PlogAppend *bool   `hcl:"log_append"`

	configFile *ConfigFile
}

type ConfigFile struct {
	*Config               `hcl:"core,block"`
	*RestApiConfig        `hcl:"api,block"`
	*ServiceInstallConfig `hcl:"service,block"`
}

func (c *Command) Help() string {
	s := bytes.NewBuffer([]byte{})
	c.Flags.SetOutput(s)
	defer c.Flags.SetOutput(os.Stderr)

	c.Flags.PrintDefaults()
	return c.SynopsisText + "\n\nUsage: " + c.HelpText + "\n" + string(s.Bytes())
}

func (c *Command) Synopsis() string {
	return c.SynopsisText
}

func (c *Command) Run(args []string) int {
	return 1
}

// Used by commands to setup the default options
func setDefaultFlags(flags *flag.FlagSet, c map[string]interface{}) {
	c["config_file"] = flags.String("config-file", "", "configuration file")
	c["debug"] = flags.Bool("debug", false, "enable debug output")
	c["level"] = flags.String("level", "", "logger output level")
	c["log_file"] = flags.String("log-file", "", "log file path")
	c["log_append"] = flags.Bool("log-append", false, "append log output to existing log file")
}

// Used by commands to process default flags and initialize the logger
func (c *Command) defaultSetup(args []string) (err error) {
	err = c.Flags.Parse(args)
	if err != nil {
		return
	}

	c.DefaultConfig = c.loadConfig()
	c.initlogger(c.DefaultConfig, c.Flags.Name())
	return
}

// Loads the default configuration values
func (c *Command) loadConfig() *Config {
	config := &Config{}
	file := &ConfigFile{}
	// Check if we have a configuration file to load and do that first
	path := c.getConfigValue("config_file", nil)
	if path != "" {
		c.loadConfigFile(path, file)
		c.DefaultConfig.configFile = file
	}

	var fc Config
	if file.Config != nil {
		fc = *file.Config
	}

	config.Debug = c.getConfigBool("debug", fc.Pdebug)
	config.Level = c.getConfigValue("level", fc.Plevel)
	config.LogFile = c.getConfigValue("log_file", fc.PlogFile)
	config.LogAppend = c.getConfigBool("log_append", fc.PlogAppend)

	return config
}

// Loads a configuration file and processes root configuration
func (c *Command) loadConfigFile(path string, config *ConfigFile) {
	f, err := os.Open(path)
	if err != nil {
		configurationError("Failed to open configuration - %s", err)
	}
	defer f.Close()

	contents, err := ioutil.ReadAll(f)
	if err != nil {
		configurationError("Failed to read configuration - %s", err)
	}

	err = hclsimple.Decode(path, contents, nil, config)
	if err != nil {
		configurationError("Failed to parse configuration - %s", err)
	}
	return
}

// Initializes the logger based on configuration values
func (c *Command) initlogger(n *Config, name string) (err error) {
	o := ioutil.Discard
	if n.Debug || n.Level != "" {
		o = os.Stderr
	}
	logOpt := &hclog.LoggerOptions{
		Name:   c.Name,
		Output: o}
	if n.LogFile != "" {
		err = os.MkdirAll(path.Dir(n.LogFile), 0755)
		if err != nil {
			return
		}
		md := os.O_CREATE | os.O_WRONLY
		if n.LogAppend {
			md = md | os.O_APPEND
		}
		f, err := os.OpenFile(n.LogFile, md, 0644)
		if err != nil {
			return err
		}
		logOpt.Output = f
	}
	if n.Debug {
		logOpt.Level = hclog.LevelFromString("trace")
	} else {
		if n.Level != "" {
			logOpt.Level = hclog.LevelFromString(n.Level)
		}
	}
	c.logger = hclog.New(logOpt)
	if name != "" {
		c.logger = c.logger.Named(name)
	}
	return
}

// Extracts value from environment variable with
// configured application prefix
func (c *Command) envName(name string) string {
	return ENV_VAR_PREFIX + strings.ToUpper(name)
}

// Gets a boolean configuration value. The current value
// is supplied from the configuration file
func (c *Command) getConfigBool(name string, current *bool) bool {
	val := *(c.flagdata[name].(*bool))
	if val && !c.isDefaultValue(name) {
		return val // cli set value
	}
	evar, ok := os.LookupEnv(c.envName(name))
	if ok && evar != "" {
		return true // env var set value
	}
	if current != nil {
		return *current // config file set value
	}
	return val // default value
}

// Gets a string configuration value. The current value
// is supplied from the configuration file
func (c *Command) getConfigValue(name string, current *string) string {
	val := *(c.flagdata[name].(*string))
	if val != "" && !c.isDefaultValue(name) {
		return val // cli set value
	}
	eval, ok := os.LookupEnv(c.envName(name))
	if ok {
		return eval // env var set value
	}
	if current != nil {
		return *current // config file set value
	}
	return val // default value
}

// Gets a string configuration value. The current value
// is supplied from the configuration file
func (c *Command) getConfigArray(name string, current []string) []string {
	val := c.getConfigValue(name, nil)
	if val == "" {
		return []string{}
	}
	return strings.Split(val, ",")
}

// Gets an int64 configuration value. The current value
// is supplied from the configuration file
func (c *Command) getConfigInt64(name string, current *int64) int64 {
	dval := *(c.flagdata[name].(*int64))
	if !c.isDefaultValue(name) {
		return dval // cli set value
	}
	eval, ok := os.LookupEnv(c.envName(name))
	if ok {
		i, err := strconv.Atoi(eval)
		if err == nil {
			return int64(i) // env var set value
		}
	}
	if current != nil {
		return *current // config file set value
	}
	return dval // default value
}

// Check if the value of a given flag is the default value
func (c *Command) isDefaultValue(name string) bool {
	name = strings.ReplaceAll(name, "_", "-")
	uset := false
	c.Flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			uset = true
		}
	})
	return !uset
}

// Used for configuration errors. It will print the provided
// error message and then force an exit
func configurationError(msg string, args ...interface{}) {
	fmt.Printf("⚠ "+msg, args...)
	panic(ForceExit{1})
}
