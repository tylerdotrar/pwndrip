package main

import (
    "flag"
    "fmt"
    "os"
    "path/filepath"

    "github.com/kgretzky/pwndrop/api"
    "github.com/kgretzky/pwndrop/config"
    "github.com/kgretzky/pwndrop/core"
    "github.com/kgretzky/pwndrop/log"
    "github.com/kgretzky/pwndrop/storage"
    "github.com/kgretzky/pwndrop/utils"

    "github.com/kgretzky/daemon"
)

const SERVICE_NAME = "pwndrop"
const SERVICE_DESCRIPTION = "pwndrop"

func usage() {
    fmt.Printf(`
Usage:
  pwndrop [subcommand] [flags]

Subcommands:
  install    Install pwndrop as a service
  remove     Remove pwndrop service
  start      Start the pwndrop service
  stop       Stop the pwndrop service
  status     Check pwndrop service status

Flags:
  -config <path>       Path to configuration file
  -debug               Enable debug output
  -no-autocert         Disable automatic certificate retrieval
  -no-dns              Disable the DNS nameserver
  -h                   Show help

Examples:
  # Flags before subcommand
  pwndrop -no-dns install
  
  # Flags after subcommand
  pwndrop install -no-dns

  # Run directly (no subcommand)
  pwndrop -debug
`)
}

// recognizedSubcommands is a map of valid subcommands
var recognizedSubcommands = map[string]bool{
    "install": true,
    "remove":  true,
    "start":   true,
    "stop":    true,
    "status":  true,
}

func main() {
    // 1) Identify if there's a recognized subcommand in os.Args
    var subcommand string
    var subcommandIndex = -1

    // We skip the first arg (the program name) and scan the rest
    for i, arg := range os.Args[1:] {
        if recognizedSubcommands[arg] {
            subcommand = arg
            // subcommandIndex is the position in the overall os.Args (not just i)
            subcommandIndex = i + 1 // +1 because i is 0-based over os.Args[1:]
            break
        }
    }

    // 2) Define our flags in the default FlagSet
    cfgPath := flag.String("config", "", "config file path")
    debugLog := flag.Bool("debug", false, "log debug output")
    disableAutocert := flag.Bool("no-autocert", false, "disable automatic certificate retrieval")
    disableDNS := flag.Bool("no-dns", false, "disable DNS nameserver")
    showHelp := flag.Bool("h", false, "show help")

    // 3) If we found a subcommand, remove it from the args list (so the flags can be anywhere)
    if subcommandIndex != -1 {
        // Build a new slice of args WITHOUT the subcommand
        newArgs := []string{os.Args[0]} // keep program name
        for i, arg := range os.Args[1:] {
            if i+1 != subcommandIndex { // skip the subcommand
                newArgs = append(newArgs, arg)
            }
        }
        // Parse the newArgs instead of default os.Args
        flag.CommandLine.Parse(newArgs[1:]) // skip program name
    } else {
        // No recognized subcommand found; just parse normally
        flag.Parse()
    }

    // 4) Create the daemon instance
    dmn, err := daemon.New(SERVICE_NAME, SERVICE_DESCRIPTION, "network.target")
    if err != nil {
        log.Error("daemon init: %s", err)
        os.Exit(1)
    }
    svc := &core.Service{Daemon: dmn}

    // 5) Check if subcommand was recognized, then handle it
    switch subcommand {
    case "install":
        installFlags := gatherInstallFlags(*disableAutocert, *disableDNS, *debugLog, *cfgPath)
        if svc.InstallWithFlags(installFlags) {
            os.Exit(0)
        } else {
            os.Exit(1)
        }

    case "remove":
        if svc.Remove() {
            os.Exit(0)
        } else {
            os.Exit(1)
        }

    case "start":
        if svc.Start() {
            os.Exit(0)
        } else {
            os.Exit(1)
        }

    case "stop":
        if svc.Stop() {
            os.Exit(0)
        } else {
            os.Exit(1)
        }

    case "status":
        if svc.Status() {
            os.Exit(0)
        } else {
            os.Exit(1)
        }
    }

    // 6) If no valid subcommand, proceed with normal "run server" logic
    if *showHelp {
        usage()
        return
    }

    if *cfgPath == "" {
        *cfgPath = utils.ExecPath("pwndrop.ini")
    }

    log.Info("pwndrop version: %s", config.Version)

    core.Cfg, err = config.NewConfig(*cfgPath)
    if err != nil {
        log.Fatal("config: %v", err)
        os.Exit(1)
    }
    api.SetConfig(core.Cfg)

    if *debugLog {
        log.SetVerbosityLevel(0)
    }

    dbPath := filepath.Join(core.Cfg.GetDataDir(), "pwndrop.db")
    log.Info("opening database at: %s", dbPath)

    storage.Open(dbPath)
    core.Cfg.HandleSetup()
    if err = core.Cfg.Save(); err != nil {
        log.Fatal("config: %v", err)
        os.Exit(1)
    }

    listenIP := core.Cfg.GetListenIP()
    log.Debug("listen_ip: %s", listenIP)
    portHTTP := core.Cfg.GetHttpPort()
    portHTTPS := core.Cfg.GetHttpsPort()

    chExit := make(chan bool, 1)
    _, err = core.NewServer(listenIP, portHTTP, portHTTPS,
        !(*disableAutocert),
        !(*disableDNS),
        &chExit)
    if err != nil {
        log.Fatal("%v", err)
        os.Exit(1)
    }

    select {
    case <-chExit:
        log.Fatal("aborting")
        os.Exit(1)
    }
}

func gatherInstallFlags(disableAutocert, disableDNS, debugLog bool, cfgPath string) []string {
    var args []string
    if cfgPath != "" {
        args = append(args, "-config", cfgPath)
    }
    if debugLog {
        args = append(args, "-debug")
    }
    if disableAutocert {
        args = append(args, "-no-autocert")
    }
    if disableDNS {
        args = append(args, "-no-dns")
    }
    return args
}
