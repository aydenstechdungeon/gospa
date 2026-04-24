package cli

// VerifyConfig controls gospa verify preflight checks.
type VerifyConfig struct {
	RoutesDir  string
	JSONOutput bool
	Quiet      bool
	Strict     bool
}

// Verify runs strict preflight checks intended for pre-dev and CI gates.
func Verify(config *VerifyConfig) {
	strict := true
	if config == nil {
		config = &VerifyConfig{}
	} else {
		strict = config.Strict
	}
	Doctor(&DoctorConfig{
		RoutesDir:  config.RoutesDir,
		JSONOutput: config.JSONOutput,
		Quiet:      config.Quiet,
		Strict:     strict,
	})
}
