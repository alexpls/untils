package constants

type Env string

const (
	EnvDev  Env = "dev"
	EnvProd Env = "prod"
)

func (e Env) String() string {
	return string(e)
}

type Mode string

const (
	ModeSelfHosted Mode = "selfhosted"
	ModeHosted     Mode = "hosted"
)

func (m Mode) String() string {
	return string(m)
}
