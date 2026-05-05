package config

import "time"

// App holds application-wide configuration knobs. Edit this file to adjust
// defaults or provide overrides via a separate file in the same package.
type App struct {
	LLM LLM
}

// LLM captures language-model-related settings.
type LLM struct {
	RequestTimeout time.Duration
}

// Settings exposes the application configuration. Users can override this by
// supplying their own init() in the config package that mutates Settings.
var Settings = App{
	LLM: LLM{
		RequestTimeout: 10 * time.Second,
	},
}
