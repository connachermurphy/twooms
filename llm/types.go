package llm

type Response struct {
	Text         string
	FinishReason string
	TokensUsed   int64
}

type Config struct {
	Model       string
	MaxTokens   int32
	Temperature float32
	System      string
}

func DefaultConfig() *Config {
	return &Config{
		Model:       "gemini-2.5-flash",
		MaxTokens:   8192,
		Temperature: 0.7,
		System:      "",
	}
}
