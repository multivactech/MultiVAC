/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"time"

	"github.com/multivactech/MultiVAC/configs/config"
	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/server"
	"github.com/multivactech/MultiVAC/server/autoupdater"
	"github.com/multivactech/MultiVAC/server/shutdown"

	_ "net/http/pprof"

	"github.com/multivactech/MultiVAC/configs/limits"
)

// winServiceMain is only invoked on Windows.  It detects when btcd is running
// as a service and reacts accordingly.
var winServiceMain func() (bool, error)

// btcdMain is the real main function for btcd.  It is necessary to work around
// the fact that deferred functions do not run when os.Exit() is called.  The
// optional serverChan parameter is mainly used by the service code to be
// notified with the server once it is setup so it can gracefully stop it when
// requested from the service control manager.
func btcdMain(serverChan chan<- *server.Server) error {
	// Load configuration and parse command line.  This function also
	// initializes logging and configures it accordingly.
	_, err := config.LoadConfig()
	defer logger.LogCleanup()
	if err != nil {
		return err
	}
	cfg := config.GlobalConfig()
	logger.InitLevel()

	// Get a channel that will be closed when a shutdown signal has been
	// triggered either from an OS signal such as SIGINT (Ctrl+C) or from
	// another subsystem such as the RPC server.
	interrupt := shutdown.InterruptListener()
	defer logger.BtcdLogger().Info("Shutdown complete")

	err = os.MkdirAll(cfg.DataDir, 0700)
	if err != nil {
		logger.BtcdLogger().Errorf("%v", err)
		return err
	}
	// Show version at startup.
	logger.BtcdLogger().Infof("Version %s", config.Version())

	// Enable http profiling server if requested.
	if cfg.Profile != "" {
		go func() {
			listenAddr := net.JoinHostPort("", cfg.Profile)
			logger.BtcdLogger().Infof("Profile server listening on %s", listenAddr)
			profileRedirect := http.RedirectHandler("/debug/pprof",
				http.StatusSeeOther)
			http.Handle("/", profileRedirect)
			logger.BtcdLogger().Errorf("%v", http.ListenAndServe(listenAddr, nil))
		}()
	}

	// Write cpu profile if requested.
	if cfg.CPUProfile != "" {
		f, err := os.Create(cfg.CPUProfile)
		if err != nil {
			logger.BtcdLogger().Errorf("Unable to create cpu profile: %v", err)
			return err
		}
		pprof.StartCPUProfile(f)
		defer f.Close()
		defer pprof.StopCPUProfile()
	}

	// Return now if an interrupt signal was triggered.
	if shutdown.InterruptRequested(interrupt) {
		return nil
	}

	// Return now if an interrupt signal was triggered.
	if shutdown.InterruptRequested(interrupt) {
		return nil
	}

	if !config.GlobalConfig().RunAsDNSSeed && !config.GlobalConfig().StorageNode {
		a, _ := time.Parse("2006-01-02 15:04:05", "2020-01-06 11:17:25")
		for time.Now().Sub(a) < 0 {
			time.Sleep(time.Second)
		}
	}

	// Create server and start it.
	server, err := server.NewServer(cfg.Listeners, config.ActiveNetParams().Params, cfg.MonitorListeners)
	if err != nil {
		// TODO: this logging could do with some beautifying.
		logger.BtcdLogger().Errorf("Unable to start server on %v: %v",
			cfg.Listeners, err)
		return err
	}
	defer func() {
		logger.BtcdLogger().Infof("Gracefully shutting down the server...")
		server.Stop()
		server.WaitForShutdown()
		logger.ServerLogger().Infof("Server shutdown complete")
	}()

	server.Start()
	if serverChan != nil {
		serverChan <- server
	}

	// Try to initiate Autoupdater
	reload := make(<-chan struct{})
	if cfg.UpdateWatchURL != "" && cfg.UpdateIntervalSec > 0 && !cfg.StorageNode && !cfg.IsOneOfFirstMiners && !cfg.RunAsDNSSeed {
		logger.BtcdLogger().Info("Start AutoUpdater.")
		autoUpdater, err := autoupdater.NewAutoUpdater(autoupdater.Options{
			BaseURL:        cfg.UpdateWatchURL,
			Architecture:   runtime.GOARCH,
			Platform:       runtime.GOOS,
			Interval:       time.Duration(cfg.UpdateIntervalSec) * time.Second,
			CurrentVersion: config.Version(),
		})
		if err != nil {
			return err
		}
		reload = autoUpdater.ShutdownChan()
		go func() {
			// Randomly sleep to avoid all client update at the same time.
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			d := r.Intn(cfg.UpdateIntervalSec)
			time.Sleep(time.Duration(d) * time.Second)
			autoUpdater.Run()
		}()
	}

	// Wait until:
	// * the interrupt signal is received from an OS signal, or
	// * shutdown is requested through one of the subsystems such as the RPC server, or
	// * autoupdater requested to reload a new version.
	select {
	case <-interrupt:
	case <-reload:
	}
	return nil
}

func main() {
	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Block and transaction processing can cause bursty allocations.  This
	// limits the garbage collector from excessively overallocating during
	// bursts.  This value was arrived at with the help of profiling live
	// usage.
	debug.SetGCPercent(10)

	// Up some limits.
	if err := limits.SetLimits(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set limits: %v\n", err)
		os.Exit(1)
	}

	// Call serviceMain on Windows to handle running as a service.  When
	// the return isService flag is true, exit now since we ran as a
	// service.  Otherwise, just fall through to normal operation.
	if runtime.GOOS == "windows" {
		isService, err := winServiceMain()
		if err != nil {
			logger.BtcdLogger().Error(err)
			os.Exit(1)
		}
		if isService {
			os.Exit(0)
		}
	}

	// Work around defer not working after os.Exit()
	if err := btcdMain(nil); err != nil {
		os.Exit(1)
	}
}
