package support

// ConfigLoader 配置加载器
type ConfigLoader interface {
	// Load 加载配置
	Load(key string) (interface{}, error)
	// LoadString 加载字符串配置
	LoadString(key string) (string, error)
	// LoadInt 加载整数配置
	LoadInt(key string) (int, error)
	// LoadBool 加载布尔配置
	LoadBool(key string) (bool, error)
}

// JsonConfig JSON 配置
type JsonConfig struct {
	data map[string]interface{}
}

// NewJsonConfig 创建 JSON 配置
func NewJsonConfig(data map[string]interface{}) *JsonConfig {
	return &JsonConfig{
		data: data,
	}
}

// Load 加载配置
func (c *JsonConfig) Load(key string) (interface{}, error) {
	return c.data[key], nil
}

// LoadString 加载字符串配置
func (c *JsonConfig) LoadString(key string) (string, error) {
	if val, ok := c.data[key].(string); ok {
		return val, nil
	}
	return "", nil
}

// LoadInt 加载整数配置
func (c *JsonConfig) LoadInt(key string) (int, error) {
	if val, ok := c.data[key].(float64); ok {
		return int(val), nil
	}
	return 0, nil
}

// LoadBool 加载布尔配置
func (c *JsonConfig) LoadBool(key string) (bool, error) {
	if val, ok := c.data[key].(bool); ok {
		return val, nil
	}
	return false, nil
}

// PropertiesConfig 属性配置
type PropertiesConfig struct {
	data map[string]string
}

// NewPropertiesConfig 创建属性配置
func NewPropertiesConfig(data map[string]string) *PropertiesConfig {
	return &PropertiesConfig{
		data: data,
	}
}

// Load 加载配置
func (c *PropertiesConfig) Load(key string) (interface{}, error) {
	return c.data[key], nil
}

// LoadString 加载字符串配置
func (c *PropertiesConfig) LoadString(key string) (string, error) {
	return c.data[key], nil
}

// LoadInt 加载整数配置
func (c *PropertiesConfig) LoadInt(key string) (int, error) {
	return 0, nil
}

// LoadBool 加载布尔配置
func (c *PropertiesConfig) LoadBool(key string) (bool, error) {
	return false, nil
}
