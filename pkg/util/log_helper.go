/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package util

import (
	"github.com/go-logr/logr"
)

type LogUtil struct {
	Log logr.Logger
}

func (l *LogUtil) LogInfo(msg string, keysAndValues ...interface{}) {
	if keysAndValues != nil {
		l.Log.Info(msg, keysAndValues)
	} else {
		l.Log.Info(msg)
	}

}

func (l *LogUtil) LogDebug(msg string, keysAndValues ...interface{}) {
	if keysAndValues != nil {
		l.Log.V(1).Info(msg, keysAndValues)
	} else {
		l.Log.V(1).Info(msg)
	}
}

func (l *LogUtil) LogError(err error, msg string, keysAndValues ...interface{}) {
	if keysAndValues != nil {
		l.Log.Error(err, msg, keysAndValues)
	} else {
		l.Log.Error(err, msg)
	}
}
