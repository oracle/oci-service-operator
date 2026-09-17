/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package script

func mockSeleniumScript(name string) string {
	return `{"id":"0d346d5f-b5e0-41b8-a739-d37ecf0f8b31","version":"2.0","name":"` + name + `","url":"https://example.com","tests":[{"id":"e2376b10-696c-4c74-aef0-78c710cedca9","name":"Mock","commands":[{"id":"d8a96cb7-044f-421a-96dd-b8731841e87e","comment":"","command":"open","target":"/","targets":[],"value":""}]}],"suites":[{"id":"84222275-96b0-4f46-a8ee-7300c78f91bd","name":"Default Suite","persistSession":false,"parallel":false,"timeout":300,"tests":["e2376b10-696c-4c74-aef0-78c710cedca9"]}],"urls":["https://example.com"],"plugins":[]}`
}
