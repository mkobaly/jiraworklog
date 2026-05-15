package config

type Args struct {
	Config  string `arg:"-c,--cfg" default:"config.yaml" help:"config file holding settings"`
	Port    int    `arg:"-p,--port" default:"8380" help:"port the web server uses"`
	Debug   bool   `arg:"-d,--debug" help:"enable verbose logging and don't send any login emails"`
	version string
}

func NewArgs(vesion string) Args {
	return Args{
		version: vesion,
	}
}

func (a Args) Version() string {
	return a.version
}

// //Define command line params and parse input
// cmdline := cmdline.New()
// cmdline.AddOption("c", "config", "config.yaml", "path to configuration file")
// cmdline.AddOption("r", "repo", "POSTGRES", "specific repo to use (MSSQL, POSTGRES)")
// cmdline.SetOptionDefault("r", "POSTGRES")
// cmdline.AddOption("p", "port", "8380", "default port to serve rest API from")
// cmdline.SetOptionDefault("p", "8380")
// cmdline.AddFlag("k", "ask", "Ask for username and password from the STDIN")
// cmdline.AddFlag("v", "verbose", "verbose logging")
// cmdline.Parse(os.Args)
