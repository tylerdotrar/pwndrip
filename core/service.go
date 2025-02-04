package core

import (
    "os"
    "path/filepath"
    "runtime"

    "github.com/kgretzky/pwndrop/log"
    "github.com/kgretzky/pwndrop/utils"

    "github.com/kgretzky/daemon"
    "github.com/otiai10/copy"
)

const (
    INSTALL_DIR = "/usr/local/pwndrop"
    EXEC_NAME   = "pwndrop"
    ADMIN_DIR   = "admin"
)

type Service struct {
    Daemon daemon.Daemon
}

func (service *Service) Install() bool {
    // For backward compatibility, just call InstallWithFlags with no flags
    return service.InstallWithFlags(nil)
}

// InstallWithFlags copies the pwndrop binaries/panel, then
// installs it as a daemon (systemd) with the provided arguments.
func (service *Service) InstallWithFlags(flags []string) bool {
    if runtime.GOOS == "windows" {
        log.Error("daemons disabled on windows")
        return false
    }

    // 1. Copy binaries & admin folder
    if !service.copyBinaries() {
        return false
    }

    // 2. Register the service with optional flags
    execDst := filepath.Join(INSTALL_DIR, EXEC_NAME)
    _, err := service.Daemon.Install(execDst, flags...) // <-- pass flags here
    if err != nil {
        if err == daemon.ErrAlreadyInstalled {
            log.Info("service already installed")
        } else {
            log.Error("failed to install daemon: %s", err)
            return false
        }
    }

    if len(flags) > 0 {
        log.Success("successfully installed daemon with flags: %v", flags)
    } else {
        log.Success("successfully installed daemon (no extra flags)")
    }
    return true
}

func (service *Service) Remove() bool {
    if runtime.GOOS == "windows" {
        log.Error("daemons disabled on windows")
        return false
    }

    _, err := service.Daemon.Remove()
    if err != nil {
        log.Error("failed to remove daemon: %s", err)
        return false
    }

    if _, err = os.Stat(INSTALL_DIR); err == nil {
        if err = os.RemoveAll(INSTALL_DIR); err != nil {
            log.Error("failed to delete directory: %s", INSTALL_DIR)
            return false
        }
        log.Success("deleted pwndrop directory")
    } else {
        log.Warning("directory not found: %s", INSTALL_DIR)
    }
    log.Success("successfully removed daemon")
    return true
}

func (service *Service) Start() bool {
    if runtime.GOOS == "windows" {
        log.Error("daemons disabled on windows")
        return false
    }
    _, err := service.Daemon.Start()
    if err != nil {
        if err == daemon.ErrAlreadyRunning {
            log.Info("daemon already running")
        } else {
            log.Error("failed to start daemon: %s", err)
            return false
        }
    }
    log.Success("pwndrop is running")
    return true
}

func (service *Service) Stop() bool {
    if runtime.GOOS == "windows" {
        log.Error("daemons disabled on windows")
        return false
    }
    _, err := service.Daemon.Stop()
    if err != nil {
        if err == daemon.ErrAlreadyStopped {
            log.Info("daemon already stopped")
        } else {
            log.Error("failed to stop daemon: %s", err)
            return false
        }
    }
    log.Success("pwndrop stopped")
    return true
}

func (service *Service) Status() bool {
    if runtime.GOOS == "windows" {
        log.Error("daemons disabled on windows")
        return false
    }
    status, err := service.Daemon.Status()
    if err != nil {
        log.Error("failed to get daemon status: %s", err)
        return false
    }
    log.Info("pwndrop status: %s", status)
    return true
}

// copyBinaries handles copying the pwndrop executable & admin folder
// to /usr/local/pwndrop.
func (service *Service) copyBinaries() bool {
    execDir := utils.GetExecDir()
    adminSrc := filepath.Join(execDir, ADMIN_DIR)

    f, err := os.Stat(adminSrc)
    if err != nil {
        log.Error("can't find admin panel in current directory: %s", adminSrc)
        return false
    }
    if !f.IsDir() {
        log.Error("'%s' is not a directory", adminSrc)
        return false
    }

    if _, err = os.Stat(INSTALL_DIR); os.IsNotExist(err) {
        if err = os.Mkdir(INSTALL_DIR, 0700); err != nil {
            log.Error("failed to create directory: %s", INSTALL_DIR)
            return false
        }
    }

    execPath, _ := os.Executable()
    execDst := filepath.Join(INSTALL_DIR, EXEC_NAME)
    if err = copy.Copy(execPath, execDst); err != nil {
        log.Error("failed to copy '%s' to: %s", execPath, execDst)
        return false
    }
    log.Success("copied pwndrop executable to: %s", execDst)

    adminDst := filepath.Join(INSTALL_DIR, ADMIN_DIR)
    if _, err = os.Stat(adminDst); err == nil {
        err = os.RemoveAll(adminDst)
        if err != nil {
            log.Error("failed to delete admin panel at: %s", adminDst)
            return false
        }
    }
    if err = copy.Copy(adminSrc, adminDst); err != nil {
        log.Error("failed to copy '%s' to: %s", adminSrc, adminDst)
        return false
    }
    log.Success("copied admin panel to: %s", adminDst)

    return true
}
