package config

type Config struct {
    Port      string
    ChromePath string
    MaxWorkers int
}

func Load() (Config, error) {
    return Config{
        Port:      ":8080",
        MaxWorkers: 50,
    }, nil // Expand with env vars or flags later
}