package main

type config struct {
	env         string
	dbUrl       string
	xAIKey      string
	openAIKey   string
	pushoverKey string
}

type serveConfig struct {
	port int
}
