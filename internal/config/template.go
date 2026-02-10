// Package config handles configuration loading and endpoint definitions
package config

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/spf13/viper"
)

// envViper is a package-level viper instance for .env file reading
var (
	envViper     *viper.Viper
	envViperOnce sync.Once
)

// initEnvViper initializes the package-level envViper instance
func initEnvViper() {
	envViperOnce.Do(func() {
		envViper = viper.New()
		envViper.SetConfigFile(".env")
		envViper.SetConfigType("env")
		// Silently ignore if .env doesn't exist
		_ = envViper.ReadInConfig()
	})
}

// getEnv retrieves an environment variable from the .env file via Viper
func getEnv(key string) string {
	initEnvViper()
	// Viper stores keys in lowercase, but we need to check both
	value := envViper.GetString(key)
	if value == "" {
		value = envViper.GetString(strings.ToLower(key))
	}
	return value
}

// TemplateFuncs provides functions for URL template evaluation
var TemplateFuncs = template.FuncMap{
	"randomString": func(length int) string {
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		b := make([]byte, length)
		for i := range b {
			b[i] = charset[rand.Intn(len(charset))]
		}
		return string(b)
	},

	"randomInt": func(min, max int) int {
		if max <= min {
			return min
		}
		return rand.Intn(max-min+1) + min
	},

	"randomPhone": func() string {
		return fmt.Sprintf("+33%d", rand.Intn(900000000)+100000000)
	},

	"randomEmail": func() string {
		return fmt.Sprintf("test%d@example.com", rand.Intn(10000))
	},

	"randomUUID": func() string {
		uuid := make([]byte, 16)
		rand.Read(uuid)
		uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
		uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant
		return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
	},

	"now": func() string {
		return time.Now().Format(time.RFC3339)
	},

	"today": func() string {
		return time.Now().Format("2006-01-02")
	},

	"yesterday": func() string {
		return time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	},

	"unixNow": func() int64 {
		return time.Now().Unix()
	},

	"unixMilli": func() int64 {
		return time.Now().UnixMilli()
	},

	"urlEncode": func(s string) string {
		return url.QueryEscape(s)
	},

	"env": func(key string) string {
		return getEnv(key)
	},

	"envDefault": func(key, defaultVal string) string {
		if val := getEnv(key); val != "" {
			return val
		}
		return defaultVal
	},
}

// TemplateData provides data for template evaluation
type TemplateData struct {
	Env map[string]string
}

// GetEnvMap returns a map of all environment variables from .env file
func GetEnvMap() map[string]string {
	initEnvViper()
	envMap := make(map[string]string)
	for _, key := range envViper.AllKeys() {
		// Viper stores keys lowercase, convert to uppercase for consistency
		upperKey := strings.ToUpper(key)
		envMap[upperKey] = envViper.GetString(key)
	}
	return envMap
}

// EvaluateTemplate evaluates a URL template with random/dynamic values
func EvaluateTemplate(templateStr string) (string, error) {
	tmpl, err := template.New("url").Funcs(TemplateFuncs).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	data := TemplateData{
		Env: GetEnvMap(),
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// EvaluateBodyTemplate evaluates a body template (for POST requests)
func EvaluateBodyTemplate(body interface{}) (interface{}, error) {
	switch v := body.(type) {
	case string:
		return EvaluateTemplate(v)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			evaluated, err := EvaluateBodyTemplate(value)
			if err != nil {
				return nil, err
			}
			result[key] = evaluated
		}
		return result, nil
	case []interface{}:
		var result []interface{}
		for _, item := range v {
			evaluated, err := EvaluateBodyTemplate(item)
			if err != nil {
				return nil, err
			}
			result = append(result, evaluated)
		}
		return result, nil
	default:
		return v, nil
	}
}

func init() {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())
}
