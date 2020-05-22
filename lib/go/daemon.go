// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package protocols

import (
	log "github.com/Sirupsen/logrus"
	"github.com/danos/vci/services"
	"sync"
	"time"
)

const (
	stopWaitSecs = 10
)

type ProtocolsDaemon struct {
	unit               string
	controlLock        sync.Mutex
	stopTimer          *time.Timer
	stopTimerDuration  time.Duration
	stopTimerStartedAt time.Time
}

func NewProtocolsDaemon(unit string) *ProtocolsDaemon {
	pd := &ProtocolsDaemon{}
	pd.unit = unit
	pd.stopTimerDuration = time.Duration(stopWaitSecs) * time.Second
	return pd
}

func (pd *ProtocolsDaemon) GetUnitName() string {
	return pd.unit
}

func (pd *ProtocolsDaemon) LockControl() {
	pd.controlLock.Lock()
}

func (pd *ProtocolsDaemon) UnlockControl() {
	pd.controlLock.Unlock()
}

func (pd *ProtocolsDaemon) Start() error {
	log.Infoln("Starting " + pd.GetUnitName())
	pd.CancelStopAndDisable()

	mgr := services.NewManager()
	defer mgr.Close()

	err := mgr.Start(pd.GetUnitName())
	if err != nil {
		log.Errorf("Failed to start %s: %s", pd.GetUnitName(), err.Error())
	}

	return err
}

func (pd *ProtocolsDaemon) Stop() error {
	log.Infoln("Stopping " + pd.GetUnitName())
	pd.CancelStopAndDisable()

	mgr := services.NewManager()
	defer mgr.Close()

	err := mgr.Stop(pd.GetUnitName())
	if err != nil {
		log.Errorf("Failed to stop %s: %s", pd.GetUnitName(), err.Error())
	}

	return err
}

func (pd *ProtocolsDaemon) Restart() error {
	log.Infoln("Restarting " + pd.GetUnitName())
	pd.CancelStopAndDisable()

	mgr := services.NewManager()
	defer mgr.Close()

	err := mgr.Restart(pd.GetUnitName())
	if err != nil {
		log.Errorf("Failed to restart %s: %s", pd.GetUnitName(), err.Error())
	}

	return err
}

func (pd *ProtocolsDaemon) Enable() error {
	log.Infoln("Enabling " + pd.GetUnitName())
	pd.CancelStopAndDisable()

	mgr := services.NewManager()
	defer mgr.Close()

	err := mgr.Enable(pd.GetUnitName())
	if err != nil {
		log.Errorf("Failed to enable %s: %s", pd.GetUnitName(), err.Error())
	}

	return err
}

func (pd *ProtocolsDaemon) Disable() error {
	log.Infoln("Disabling " + pd.GetUnitName())
	pd.CancelStopAndDisable()

	mgr := services.NewManager()
	defer mgr.Close()

	err := mgr.Disable(pd.GetUnitName())
	if err != nil {
		log.Errorf("Failed to disable %s: %s", pd.GetUnitName(), err.Error())
	}

	return err
}

func (pd *ProtocolsDaemon) stopAndDisableCallback() {
	pd.LockControl()
	defer pd.UnlockControl()

	/* Check there wasn't a late timer cancellation */
	if pd.stopTimer == nil {
		return
	}

	/* Check timer wasn't rescheduled after a previous one fired */
	if time.Now().After(pd.stopTimerStartedAt.Add(pd.stopTimerDuration)) {
		pd.Stop()
		pd.Disable()
		pd.stopTimer = nil
	}
}

func (pd *ProtocolsDaemon) StopAndDisableIfScheduled() {
	if pd.CancelStopAndDisable() {
		pd.Stop()
		pd.Disable()
	}
}

func (pd *ProtocolsDaemon) ScheduleStopAndDisable() {
	log.Infof("Scheduling %s to be stopped in %v secs", pd.GetUnitName(), pd.stopTimerDuration.Seconds())
	pd.CancelStopAndDisable()
	pd.stopTimerStartedAt = time.Now()
	pd.stopTimer = time.AfterFunc(pd.stopTimerDuration, pd.stopAndDisableCallback)
}

func (pd *ProtocolsDaemon) CancelStopAndDisable() bool {
	if pd.stopTimer != nil {
		if pd.stopTimer.Stop() {
			log.Infof("Cancelled scheduled stop of %s", pd.GetUnitName())
		}
		pd.stopTimer = nil
		return true
	}

	return false
}
