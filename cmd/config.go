package main

type config struct {
	buildVersion string
	env          string
	dbUrl        string
	demoUserID   int64
	xAIKey       string
	openAIKey    string
	braveKey     string
	pushoverKey  string
	smtp         struct {
		username string
		password string
		host     string
		port     int
	}
}

type serveConfig struct {
	port int
}
