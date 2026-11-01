package envfile

import (
	"os"
	"github.com/joho/godotenv"
)

func GetEnvs(envFile string) (map[string]string, error) {
	file, err := os.Open(envFile)
	if err != nil {
		return nil, err
	}

	return godotenv.Parse(file)
}
