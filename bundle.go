package termboxScreen

type Bundle map[string]interface{}

func (b Bundle) SetValue(key string, val interface{}) {
	b[key] = val
}

func (b Bundle) GetBool(key string, def bool) bool {
	if v, ok := b[key].(bool); ok {
		return v
	}
	return def
}

func (b Bundle) GetString(key, def string) string {
	if v, ok := b[key].(string); ok {
		return v
	}
	return def
}

func (b Bundle) GetInt(key string, def int) int {
	if v, ok := b[key].(int); ok {
		return v
	}
	return def
}
