/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package template

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"testing"
)

func mockTemplateZip(t *testing.T) string {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create("main.tf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("terraform { required_version = \">= 1.0\" }\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes())
}
