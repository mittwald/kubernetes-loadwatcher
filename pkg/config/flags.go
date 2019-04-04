package config

type StartupFlags struct {
	KubeConfig     string
	TaintThreshold int
	EvictThreshold int
	EvictBackoff   string
	NodeName       string
}
