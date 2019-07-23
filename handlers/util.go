package handlers

func filter(scs []SowerConfig, test func(SowerConfig) bool) (ret []SowerConfig) {
	for _, s := range scs {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}
