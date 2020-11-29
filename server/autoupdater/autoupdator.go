package autoupdater

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
)

const (
	defaultInterval = 10 * time.Minute
)

/*type rebooter struct {
}*/

// Options options to create new AutoUpdater
type Options struct {
	// The path of the target to be updated.
	// If it's left empty, the path of the current executable will be used.
	// This is mainly for testing purpose.
	Target string
	// The base URL for fetch update info.
	BaseURL string
	// The architecture (e.g. x86, x86_64) this program runs on.
	// Will be used for composing the update URL.
	Architecture string
	// The platform/os (e.g. linux, darwin) this program runs on.
	// Will be used for composing the update URL.
	Platform string

	CurrentVersion string

	Interval time.Duration
}

// AutoUpdater handles automatic updating.
type AutoUpdater struct {
	target         string
	poller         *poller
	currentVersion string
	shutdown       chan struct{}
	logger         btclog.Logger
}

// NewAutoUpdater creates new AutoUpdater and initialize it.
func NewAutoUpdater(options Options) (*AutoUpdater, error) {
	target := options.Target
	if target == "" {
		var err error
		target, err = os.Executable()
		if err != nil {
			log.Fatal("Cannot find executable path: ", err)
		}
	}

	interval := options.Interval
	if interval == 0 {
		interval = defaultInterval
	}

	updateURL, err := url.Parse(
		fmt.Sprintf("%s?architecture=%s&os=%s",
			options.BaseURL, options.Architecture, options.Platform))
	if err != nil {
		return nil, fmt.Errorf("Cannot parse baseUrl: %v", options.BaseURL)
	}

	poller := newPoller(*updateURL, interval)

	return &AutoUpdater{
		target:         target,
		poller:         poller,
		currentVersion: options.CurrentVersion,
		shutdown:       make(chan struct{}),
		logger:         logger.ServerLogger(),
	}, nil
}

// VersionInfo contain information about a version of the software.
// This must be kept sync with the `DBVersion` struct in the browser repo.
type VersionInfo struct {
	VersionID   string   `json:"VersionID"`
	Tips        []string `json:"Tips"`
	DownloadURL string   `json:"DownloadURL"`
}

// Run starts polling and wait for updates.
// This method is block. Consider run it in a gorutine.
func (a *AutoUpdater) Run() {
	a.logger.Infof("Current version is %s, wait for update.\n", a.currentVersion)
	a.poller.Start()
	for change := range a.poller.changes {
		newBinary, err := a.handleChange(change)
		if err != nil {
			a.logger.Error(err)
			continue
		}
		if newBinary != nil {
			err := a.Apply(bytes.NewReader(newBinary))
			if err != nil {
				a.logger.Error(err)
				continue
			}
			err = a.reload()
			if err != nil {
				a.logger.Error(err)
				continue
			}
		}
	}
	a.logger.Info("Poller is stoped. Stop running AutoUpdater.")
}

func (a *AutoUpdater) handleChange(change []byte) ([]byte, error) {
	var versionInfo VersionInfo
	err := json.Unmarshal(change, &versionInfo)
	if err != nil {
		return nil, fmt.Errorf("invalid version info %s: %s", change, err)
	}
	if versionInfo.VersionID == a.currentVersion {
		return nil, nil
	}
	newBinary, err := download(versionInfo.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download new version: %s", err)
	}
	return newBinary, nil
}

type rollbackErr struct {
	error
	rollbackErr error
}

// Apply replaces the software binary with new version, and keep an backup of the old one.
func (a *AutoUpdater) Apply(update io.Reader) error {
	if strings.Compare(a.target, "") == 0 {
		return fmt.Errorf("Empty target path")
	}
	updateDir := filepath.Dir(a.target)
	fileNmae := filepath.Base(a.target)
	newPath := filepath.Join(updateDir, fmt.Sprintf(".%s.new", fileNmae))
	oldPath := filepath.Join(updateDir, fmt.Sprintf(".%s.old", fileNmae))
	fp, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer fp.Close()

	_, err = io.Copy(fp, update)
	if err != nil {
		return err
	}
	fp.Close()

	_ = os.Remove(oldPath)

	err = os.Rename(a.target, oldPath)
	if err != nil {
		return err
	}

	err = os.Rename(newPath, a.target)
	if err != nil {
		// move unsuccessful
		// Need to rollback the change.
		rerr := os.Rename(oldPath, a.target)
		if rerr != nil {
			return &rollbackErr{err, rerr}
		}
		return err
	}
	return nil
}

func (a *AutoUpdater) reload() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	execSpec := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	}
	fork, err := syscall.ForkExec(execPath, os.Args, execSpec)
	if err != nil {
		return err
	}
	a.logger.Info("reloaded to new process ", fork)
	a.shutdown <- struct{}{}
	return nil
}

// ShutdownChan Returns a channel to indicate the server should shutdown.
// This usually happens when the new version is ready.
func (a *AutoUpdater) ShutdownChan() <-chan struct{} {
	return a.shutdown
}
