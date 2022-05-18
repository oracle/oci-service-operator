/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loggerutil

import (
	"context"
	"github.com/pkg/errors"
	"strings"
)

const FixedLogMapCtxKey = LogMapCtxKey("fixedLogMap")

func extractKeyValuePairs(keyValues []interface{}) (string, error) {

	if len(keyValues) == 0 {
		return "", nil
	}
	fields := make([]string, 0, len(keyValues)/2)
	for index := 0; index < len(keyValues); {
		key, keyIsString := keyValues[index].(string)
		value, valueIsString := keyValues[index+1].(string)

		if !keyIsString || !valueIsString {
			return "", errors.New("key and value must be string")
		}

		fields = append(fields, key+": "+value)
		index += 2
	}
	return strings.Join(fields, " , "), nil
}

func finalMessageBuilder(message string, fixedMessage string, extraParameters string) string {
	buildingMessage := ""

	if len(message) != 0 {
		buildingMessage = " { " + " Message: " + message
	}

	if len(buildingMessage) != 0 && len(extraParameters) != 0 {
		buildingMessage = buildingMessage + " , " + extraParameters
	} else if len(extraParameters) != 0 {
		buildingMessage = "{ " + extraParameters
	}

	if len(buildingMessage) != 0 && len(fixedMessage) != 0 {
		buildingMessage = buildingMessage + " , " + fixedMessage
	} else if len(fixedMessage) != 0 {
		buildingMessage = "{ " + fixedMessage
	}

	if len(buildingMessage) != 0 {
		buildingMessage += " } "
	}

	return buildingMessage
}

func fixedMessageBuilder(ctx context.Context) string {
	if ctx != nil && ctx.Value(FixedLogMapCtxKey) != nil {
		fixedLogMap, ok := ctx.Value(FixedLogMapCtxKey).(map[string]string)
		if ok && len(fixedLogMap) != 0 {
			fixedMessageArray := make([]string, 0, len(fixedLogMap))

			for key, value := range fixedLogMap {
				entry := key + ": " + value
				fixedMessageArray = append(fixedMessageArray, entry)
			}

			return strings.Join(fixedMessageArray, " , ")
		}
	}
	return ""
}
