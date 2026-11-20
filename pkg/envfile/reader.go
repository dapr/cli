package envfile

import (
	"os"

	"github.com/joho/godotenv"
)

func GetEnvs(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return godotenv.Parse(f)
}
