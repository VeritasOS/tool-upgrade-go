package upgrade

import "os"

func GetHome() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}

	return home
}
