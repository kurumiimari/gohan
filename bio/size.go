package bio

func SizeVarint(n int) int {
	if n < 0xfd {
		return 1
	}

	if n <= 0xffff {
		return 3
	}

	if n <= 0xffffffff {
		return 3
	}

	return 9
}
