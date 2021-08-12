/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loggerutil

import (
	"github.com/go-logr/logr"
	"strings"
)

type OSOKLogger struct {
	Logger    logr.Logger
	FixedLogs map[string]string
}

func (ol *OSOKLogger) DebugLog(message string, keysAndValues ...interface{}) {

	fixedMessage := ""

	if len(ol.FixedLogs) != 0 {
		fixedMessage = fixedMessageBuilder(ol)
	}

	res, err := extractKeyValuePairs(keysAndValues)
	if err != nil {
		ol.Logger.Error(err, "Passed Key value are not string only string allowed")
		return
	}

	finalMessage := finalMessageBuilder(message, fixedMessage, res)
	if len(finalMessage) == 0 {
		return
	}

	ol.Logger.V(1).Info(finalMessage)
}

func (ol *OSOKLogger) InfoLog(message string, keysAndValues ...interface{}) {

	fixedMessage := ""

	if len(ol.FixedLogs) != 0 {
		fixedMessage = fixedMessageBuilder(ol)
	}

	res, err := extractKeyValuePairs(keysAndValues)
	if err != nil {
		ol.Logger.Error(err, "Passed Key value are not string only string allowed")
		return
	}

	finalMessage := finalMessageBuilder(message, fixedMessage, res)
	if len(finalMessage) == 0 {
		return
	}

	ol.Logger.Info(finalMessage)
}

func (ol *OSOKLogger) ErrorLog(err error, message string, keysAndValues ...interface{}) {

	fixedMessage := ""

	if len(ol.FixedLogs) != 0 {
		fixedMessageArray := make([]string, 0, len(ol.FixedLogs))

		for key, value := range ol.FixedLogs {
			entry := key + ": " + value
			fixedMessageArray = append(fixedMessageArray, entry)
		}

		fixedMessage = strings.Join(fixedMessageArray, " , ")
	}

	res, err := extractKeyValuePairs(keysAndValues)
	if err != nil {
		ol.Logger.Error(err, "Passed Key value are not string only string allowed")
		return
	}

	finalMessage := finalMessageBuilder(message, fixedMessage, res)
	if len(finalMessage) == 0 {
		return
	}

	ol.Logger.Error(err, finalMessage)
}
