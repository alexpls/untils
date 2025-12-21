package main

type config struct {
	env         string
	dbUrl       string
	xAIKey      string
	openAIKey   string
	pushoverKey string
	smtp        struct {
		username string
		password string
		host     string
		port     int
	}
}

type serveConfig struct {
	port int
}
