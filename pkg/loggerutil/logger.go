/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loggerutil

import (
	"context"
	"github.com/go-logr/logr"
)

type OSOKLogger struct {
	Logger logr.Logger
}

type LogMapCtxKey string

func (ol *OSOKLogger) DebugLog(message string, keysAndValues ...interface{}) {
	res, err := extractKeyValuePairs(keysAndValues)
	if err != nil {
		ol.Logger.Error(err, "Passed Key value are not string only string allowed")
		return
	}

	finalMessage := finalMessageBuilder(message, "", res)
	if len(finalMessage) == 0 {
		return
	}

	ol.Logger.V(1).Info(finalMessage)
}

func (ol *OSOKLogger) InfoLog(message string, keysAndValues ...interface{}) {
	res, err := extractKeyValuePairs(keysAndValues)
	if err != nil {
		ol.Logger.Error(err, "Passed Key value are not string only string allowed")
		return
	}

	finalMessage := finalMessageBuilder(message, "", res)
	if len(finalMessage) == 0 {
		return
	}

	ol.Logger.Info(finalMessage)
}

func (ol *OSOKLogger) ErrorLog(err error, message string, keysAndValues ...interface{}) {
	res, extractErr := extractKeyValuePairs(keysAndValues)
	if extractErr != nil {
		ol.Logger.Error(extractErr, "Passed Key value are not string only string allowed")
		return
	}

	finalMessage := finalMessageBuilder(message, "", res)
	if len(finalMessage) == 0 {
		return
	}

	ol.Logger.Error(err, finalMessage)
}

func (ol *OSOKLogger) DebugLogWithFixedMessage(ctx context.Context, message string, keysAndValues ...interface{}) {

	fixedMessage := fixedMessageBuilder(ctx)

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

func (ol *OSOKLogger) InfoLogWithFixedMessage(ctx context.Context, message string, keysAndValues ...interface{}) {

	fixedMessage := fixedMessageBuilder(ctx)

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

func (ol *OSOKLogger) ErrorLogWithFixedMessage(ctx context.Context, err error, message string, keysAndValues ...interface{}) {

	fixedMessage := fixedMessageBuilder(ctx)

	res, extractErr := extractKeyValuePairs(keysAndValues)
	if extractErr != nil {
		ol.Logger.Error(extractErr, "Passed Key value are not string only string allowed")
		return
	}

	finalMessage := finalMessageBuilder(message, fixedMessage, res)
	if len(finalMessage) == 0 {
		return
	}

	ol.Logger.Error(err, finalMessage)
}
