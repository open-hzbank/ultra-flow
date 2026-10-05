package core

// Env 统一环境定义
type Env string

const (
	EnvDev  Env = "dev"  // 开发环境
	EnvSit  Env = "sit"  // 集成测试环境
	EnvUat  Env = "uat"  // 验收测试环境
	EnvProd Env = "prod" // 生产环境
)

var nonProductionEnvs = map[Env]bool{EnvDev: true, EnvSit: true, EnvUat: true}

func (e Env) IsNonProduction() bool { return nonProductionEnvs[e] }
func (e Env) IsProduction() bool    { return !nonProductionEnvs[e] }

func NonProductionEnvs() []Env { return []Env{EnvDev, EnvSit, EnvUat} }

func EnvOfName(name string) (Env, bool) {
	switch name {
	case "dev":
		return EnvDev, true
	case "sit":
		return EnvSit, true
	case "uat":
		return EnvUat, true
	case "prod":
		return EnvProd, true
	default:
		return "", false
	}
}
