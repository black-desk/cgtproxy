package internal

import "os/exec"

func GetNFTableRules() string {
	out, err := exec.Command("nft", "list", "ruleset").Output()
	if err != nil {
		panic(err)
	}

	return string(out)
}
